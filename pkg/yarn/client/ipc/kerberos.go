/*
Copyright 2013 The Cloudera Inc.
Copyright 2023 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ipc

import (
	"fmt"
	"os"
	"strings"

	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/gssapi"
	"github.com/jcmturner/gokrb5/v8/iana/chksumtype"
	"github.com/jcmturner/gokrb5/v8/iana/nametype"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/messages"
	"github.com/jcmturner/gokrb5/v8/types"
	"k8s.io/klog/v2"

	hadoop_common "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
)

const (
	EnvEnableKerberos = "ENABLE_KERBEROS"
	EnvKrb5ConfPath   = "KRB5_CONF_PATH"
	EnvKeytabPath     = "KRB5_KEYTAB_PATH"
	EnvClientUserName = "KRB5_CLIENT_USERNAME"
	EnvClientRealm    = "KRB5_CLIENT_REALM"

	DefaultKrb5ConfPath   = "/etc/krb5.conf"
	DefaultKeytabPath     = "/etc/security/keytabs/yarn.keytab"
	DefaultClientUserName = "yarn-operator"
	DefaultRealm          = "EXAMPLE.COM"
)

type KerberosConf struct {
	Krb5ConfPath   string
	KeytabPath     string
	ClientUserName string
	ClientRealm    string
}

func IsEnableKerberos() bool {
	if v, ok := os.LookupEnv(EnvEnableKerberos); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "y", "on", "TRUE":
			return true
		case "0", "false", "no", "n", "off", "FALSE":
			return false
		}
	}
	return false
}

func GetKerberosConf() KerberosConf {
	conf := KerberosConf{
		Krb5ConfPath:   DefaultKrb5ConfPath,
		KeytabPath:     DefaultKeytabPath,
		ClientUserName: DefaultClientUserName,
		ClientRealm:    DefaultRealm,
	}

	if v, ok := os.LookupEnv(EnvKrb5ConfPath); ok && v != "" {
		conf.Krb5ConfPath = v
	}

	if v, ok := os.LookupEnv(EnvKeytabPath); ok && v != "" {
		conf.KeytabPath = v
	}

	if v, ok := os.LookupEnv(EnvClientUserName); ok && v != "" {
		conf.ClientUserName = v
	}

	if v, ok := os.LookupEnv(EnvClientRealm); ok && v != "" {
		conf.ClientRealm = v
	}

	return conf
}

func negotiateKerberosAuth(client *Client, con *connection) error {
	// SASL NEGOTIATE
	negState := hadoop_common.RpcSaslProto_NEGOTIATE
	if err := sendSaslMessage(client, con, &hadoop_common.RpcSaslProto{State: &negState}); err != nil {
		return err
	}

	resp, err := receiveSaslMessage(client, con)
	if err != nil {
		return err
	}

	var krbAuth *hadoop_common.RpcSaslProto_SaslAuth
	for i, a := range resp.Auths {
		klog.Infof("Server support auth [%d]: Method=%s, Mechanism=%s, Protocol=%s, ServerID=%s", i, a.GetMethod(), a.GetMechanism(), a.GetProtocol(), a.GetServerId())
		if a.GetMethod() == "KERBEROS" && a.GetMechanism() == "GSSAPI" {
			krbAuth = a
		}
	}
	if krbAuth == nil {
		klog.Errorf("kerberos with gssapi is not supported")
		return fmt.Errorf("kerberos with gssapi is not supported")
	}

	// Kerberos Handshake
	spn := fmt.Sprintf("%s/%s", krbAuth.GetProtocol(), krbAuth.GetServerId())
	krb5Conf := GetKerberosConf()
	token, sessionKey, err := generateGssapiKerberosToken(krb5Conf, spn)
	if err != nil {
		klog.Errorf("failed to generate gss-api kerberos token: %v", err)
		return err
	}

	// SASL INITIATE
	saslInitiateState := hadoop_common.RpcSaslProto_INITIATE
	saslInitiateMessage := hadoop_common.RpcSaslProto{
		State: &saslInitiateState,
		Token: token,
		Auths: []*hadoop_common.RpcSaslProto_SaslAuth{krbAuth},
	}

	if err = sendSaslMessage(client, con, &saslInitiateMessage); err != nil {
		klog.Errorf("failed to send sasl initiate message: %v", err)
		return err
	}

	saslResponseMessage, err := receiveSaslMessage(client, con)
	if err != nil {
		klog.Errorf("failed to receive sasl message: %v", err)
		return err
	}

	// SASL CHALLENGE
	if saslResponseMessage.GetState() == hadoop_common.RpcSaslProto_CHALLENGE {
		// check hadoop qop
		var serverWrapToken gssapi.WrapToken
		if err := serverWrapToken.Unmarshal(saslResponseMessage.GetToken(), true); err != nil {
			klog.Errorf("failed to unmarshal server challenge token: %v", err)
			return err
		}
		serverQOP := serverWrapToken.Payload[0]
		if serverWrapToken.Payload[0] > 0x01 {
			klog.Errorf("Server requires QOP %d (Integrity/Privacy), but this client only supports 1 (Authentication)", serverQOP)
			return fmt.Errorf("unsupported SASL QOP level: %d", serverQOP)
		}

		rawPayload := []byte{0x01, 0x00, 0x00, 0x00} // require hadoop qop Authentication
		authzID := []byte(fmt.Sprintf("%s@%s", krb5Conf.ClientUserName, krb5Conf.ClientRealm))
		fullPayload := append(rawPayload, authzID...)

		wrapToken, err := gssapi.NewInitiatorWrapToken(fullPayload, *sessionKey)
		if err != nil {
			klog.Errorf("failed to wrap challenge token: %v", err)
			return err
		}

		finalBytes, err := wrapToken.Marshal()
		if err != nil {
			klog.Errorf("failed to marshal challenge token: %v", err)
			return err
		}

		// SASL RESPONSE
		saslResponseState := hadoop_common.RpcSaslProto_RESPONSE
		saslResponseMessageToSend := hadoop_common.RpcSaslProto{
			State: &saslResponseState,
			Token: finalBytes,
		}

		if err = sendSaslMessage(client, con, &saslResponseMessageToSend); err != nil {
			klog.Errorf("failed to send sasl response message: %v", err)
			return err
		}

		saslResponseMessage, err = receiveSaslMessage(client, con)
		if err != nil {
			klog.Errorf("failed to receive sasl message: %v", err)
			return err
		}
	}

	// SASL SUCCESS
	if saslResponseMessage.GetState() != hadoop_common.RpcSaslProto_SUCCESS {
		return fmt.Errorf("kerberos SASL auth failed at server side, expect 0 acutal %d", saslResponseMessage.GetState())
	}

	klog.Infof("Successfully completed Kerberos SASL negotiation")
	return nil
}

func generateGssapiKerberosToken(krb5Config KerberosConf, spn string) ([]byte, *types.EncryptionKey, error) {
	krb5Conf, err := config.Load(krb5Config.Krb5ConfPath)
	if err != nil {
		klog.Errorf("failed to load krb5 conf: %v", err)
		return nil, nil, err
	}

	keytab, err := keytab.Load(krb5Config.KeytabPath)
	if err != nil {
		klog.Errorf("failed to load keytab: %v", err)
		return nil, nil, err
	}

	kerberosClient := client.NewWithKeytab(krb5Config.ClientUserName, krb5Config.ClientRealm, keytab, krb5Conf)

	if err := kerberosClient.Login(); err != nil {
		klog.Errorf("failed to login: %v", err)
		return nil, nil, err
	}

	targetSPN := types.NewPrincipalName(nametype.KRB_NT_PRINCIPAL, spn)
	serverTicket, sessionKey, err := kerberosClient.GetServiceTicket(targetSPN.PrincipalNameString())
	if err != nil {
		klog.Errorf("failed to get service ticket: %v", err)
		return nil, nil, err
	}

	auth, err := types.NewAuthenticator(kerberosClient.Credentials.Realm(), kerberosClient.Credentials.CName())
	if err != nil {
		klog.Errorf("failed to get authenticator: %v", err)
		return nil, nil, err
	}
	gssCksum := make([]byte, 24)
	gssCksum[0] = 0x10
	gssCksum[20] = 0x00
	auth.Cksum = types.Checksum{
		CksumType: chksumtype.GSSAPI,
		Checksum:  gssCksum,
	}

	apReq, err := messages.NewAPReq(serverTicket, sessionKey, auth)
	if err != nil {
		klog.Errorf("failed to create AP_REQ: %v", err)
		return nil, nil, err
	}
	apReqBytes, err := apReq.Marshal()
	if err != nil {
		klog.Errorf("failed to marshal AP_REQ: %v", err)
		return nil, nil, err
	}

	// GSSAPI Encapsulation
	oid := []byte{0x06, 0x09, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x12, 0x01, 0x02, 0x02}
	tokenID := []byte{0x01, 0x00}

	payload := append(oid, tokenID...)
	payload = append(payload, apReqBytes...)

	// Build Application Construct Tag (0x60)
	var finalToken []byte
	finalToken = append(finalToken, 0x60)
	finalToken = append(finalToken, encodeASN1Length(len(payload))...)
	finalToken = append(finalToken, payload...)

	return finalToken, &sessionKey, nil
}

func encodeASN1Length(length int) []byte {
	if length <= 127 {
		return []byte{byte(length)}
	}
	if length <= 255 {
		return []byte{0x81, byte(length)}
	}
	if length <= 65535 {
		return []byte{0x82, byte(length >> 8), byte(length & 0xFF)}
	}
	return []byte{0x83, byte(length >> 16), byte(length >> 8 & 0xFF), byte(length & 0xFF)}
}
