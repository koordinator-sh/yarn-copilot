package ipc

import (
	"encoding/binary"
	"os"
	"strings"

	"github.com/jcmturner/gofork/encoding/asn1"
	"github.com/jcmturner/gokrb5/v8/asn1tools"
	krb5client "github.com/jcmturner/gokrb5/v8/client"
	krb5config "github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/gssapi"
	"github.com/jcmturner/gokrb5/v8/iana/chksumtype"
	"github.com/jcmturner/gokrb5/v8/iana/keyusage"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/messages"
	"github.com/jcmturner/gokrb5/v8/types"

	"k8s.io/klog/v2"
)

type GSSAPIKerberosAuth struct {
	ticket   messages.Ticket
	encKey   types.EncryptionKey
	username string
	realm    string
	cname    types.PrincipalName
	step     int
}

const DefaultKrb5Config = "/etc/krb5.conf"

func CreateKerberosClient(keytabFilePath, principal string) *GSSAPIKerberosAuth {
	krb5ConfigFile := os.Getenv("KRB5_CONFIG")
	if krb5ConfigFile == "" {
		krb5ConfigFile = DefaultKrb5Config
	}
	cfg, err := krb5config.Load(krb5ConfigFile)
	if err != nil {
		klog.Errorf("failed to load krb5 config: %v", err)
		return nil
	}

	kt, err := keytab.Load(keytabFilePath)
	if err != nil {
		klog.Error("failed to load keytab: %v", err)
		return nil
	}

	// yarn.resourcemanager.principal
	// eg: yarn/master.example.com@EXAMPLE.COM
	parts := strings.Split(principal, "@")
	username, realm := parts[0], parts[1]
	krbclient := krb5client.NewWithKeytab(username, realm, kt, cfg)

	if err := krbclient.Login(); err != nil {
		klog.Errorf("kerberos client error: %v", err)
		return nil
	}

	ticket, encKey, err := krbclient.GetServiceTicket(username)
	if err != nil {
		klog.Errorf("error getting Kerberos service ticket : %v", err)
		return nil
	}

	kerberosClient := &GSSAPIKerberosAuth{
		username: username,
		realm:    krbclient.Credentials.Domain(),
		cname:    krbclient.Credentials.CName(),

		ticket: ticket,
		encKey: encKey,
	}
	// Set the initial step to GSS_API_INITIAL
	kerberosClient.step = GSS_API_INITIAL
	return kerberosClient
}

// copy from https://github.com/IBM/sarama/sarama/kerberos_client.go initSecContext
func (krbAuth *GSSAPIKerberosAuth) EvaluateChallenge(token []byte) ([]byte, error) {
	switch krbAuth.step {
	case GSS_API_INITIAL:
		aprBytes, err := krbAuth.createKrb5Token(krbAuth.realm, krbAuth.cname, krbAuth.ticket, krbAuth.encKey)
		if err != nil {
			return nil, err
		}

		krbAuth.step = GSS_API_VERIFY
		return krbAuth.appendGSSAPIHeader(aprBytes)
	case GSS_API_VERIFY:
		wrapTokenReq := gssapi.WrapToken{}
		if err := wrapTokenReq.Unmarshal(token, true); err != nil {
			return nil, err
		}
		// Validate response
		isValid, err := wrapTokenReq.Verify(krbAuth.encKey, keyusage.GSSAPI_ACCEPTOR_SEAL)
		if !isValid {
			return nil, err
		}

		wrapTokenResponse, err := gssapi.NewInitiatorWrapToken(wrapTokenReq.Payload, krbAuth.encKey)
		if err != nil {
			return nil, err
		}
		krbAuth.step = GSS_API_FINISH
		return wrapTokenResponse.Marshal()
	}
	return nil, nil
}

// NOTE: copy from https://github.com/IBM/sarama/blob/main/gssapi_kerberos.go
const (
	TOK_ID_KRB_AP_REQ   = 256
	GSS_API_GENERIC_TAG = 0x60

	KRB5_USER_AUTH   = 1
	KRB5_KEYTAB_AUTH = 2
	KRB5_CCACHE_AUTH = 3

	GSS_API_INITIAL = 1
	GSS_API_VERIFY  = 2
	GSS_API_FINISH  = 3
)

func (krbAuth *GSSAPIKerberosAuth) newAuthenticatorChecksum() []byte {
	a := make([]byte, 24)
	flags := []int{gssapi.ContextFlagInteg, gssapi.ContextFlagConf}
	binary.LittleEndian.PutUint32(a[:4], 16)
	for _, i := range flags {
		f := binary.LittleEndian.Uint32(a[20:24])
		f |= uint32(i)
		binary.LittleEndian.PutUint32(a[20:24], f)
	}
	return a
}

/*
*
* Construct Kerberos AP_REQ package, conforming to RFC-4120
* https://tools.ietf.org/html/rfc4120#page-84
*
 */
func (krbAuth *GSSAPIKerberosAuth) createKrb5Token(
	domain string, cname types.PrincipalName,
	ticket messages.Ticket,
	sessionKey types.EncryptionKey) ([]byte, error) {
	auth, err := types.NewAuthenticator(domain, cname)
	if err != nil {
		return nil, err
	}
	auth.Cksum = types.Checksum{
		CksumType: chksumtype.GSSAPI,
		Checksum:  krbAuth.newAuthenticatorChecksum(),
	}
	APReq, err := messages.NewAPReq(
		ticket,
		sessionKey,
		auth,
	)
	if err != nil {
		return nil, err
	}
	aprBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(aprBytes, TOK_ID_KRB_AP_REQ)
	tb, err := APReq.Marshal()
	if err != nil {
		return nil, err
	}
	aprBytes = append(aprBytes, tb...)
	return aprBytes, nil
}

/*
*
*	Append the GSS-API header to the payload, conforming to RFC-2743
*	Section 3.1, Mechanism-Independent Token Format
*
*	https://tools.ietf.org/html/rfc2743#page-81
*
*	GSSAPIHeader + <specific mechanism payload>
*
 */
func (krbAuth *GSSAPIKerberosAuth) appendGSSAPIHeader(payload []byte) ([]byte, error) {
	oidBytes, err := asn1.Marshal(gssapi.OIDKRB5.OID())
	if err != nil {
		return nil, err
	}
	tkoLengthBytes := asn1tools.MarshalLengthBytes(len(oidBytes) + len(payload))
	GSSHeader := append([]byte{GSS_API_GENERIC_TAG}, tkoLengthBytes...)
	GSSHeader = append(GSSHeader, oidBytes...)
	GSSPackage := append(GSSHeader, payload...)
	return GSSPackage, nil
}
