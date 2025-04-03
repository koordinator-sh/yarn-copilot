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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	gouuid "github.com/nu7hatch/gouuid"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"k8s.io/klog/v2"

	yarnauth "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/auth"
	hadoop_common "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/security"
)

const (
	connDefaultTimeout = 10 * time.Second
	rwDefaultTimeout   = 5 * time.Second
)

var (
	saslNegotiateState   = hadoop_common.RpcSaslProto_NEGOTIATE
	saslNegotiateMessage = hadoop_common.RpcSaslProto{
		State: &saslNegotiateState,
	}
)

type Client struct {
	ClientId      *gouuid.UUID
	UGI           *security.UserGroupInformation
	ServerAddress string

	TCPNoDelay    bool
	TCPLowLatency bool
}

type connection struct {
	con *net.TCPConn
}

type connection_id struct {
	protocol string
	address  string
	ClientId gouuid.UUID
}

type call struct {
	callId    int32
	procedure proto.Message
	request   proto.Message
	response  proto.Message
	// err        *error
	retryCount int32
}

func (c *Client) String() string {
	buf := bytes.NewBufferString("")
	fmt.Fprint(buf, "<clientId:", c.ClientId)
	fmt.Fprint(buf, ", server:", c.ServerAddress)
	fmt.Fprint(buf, ">")
	return buf.String()
}

var (
	SASL_RPC_DUMMY_CLIENT_ID     []byte = make([]byte, 0)
	SASL_RPC_CALL_ID             int32  = -33
	SASL_RPC_INVALID_RETRY_COUNT int32  = -1

	AUTHORIZATION_FAILED_CALL_ID int32 = -1
	INVALID_CALL_ID              int32 = -2
	CONNECTION_CONTEXT_CALL_ID   int32 = -3
	PING_CALL_ID                 int32 = -4
)

func (c *Client) Call(rpc *hadoop_common.RequestHeaderProto, rpcRequest proto.Message, rpcResponse proto.Message) error {
	// Create connection_id
	connectionId := connection_id{
		protocol: *rpc.DeclaringClassProtocolName,
		address:  c.ServerAddress,
		ClientId: *c.ClientId,
	}

	// Get connection to server
	klog.V(5).Infof("Connecting... %v", c)
	conn, err := getConnection(c, &connectionId)
	if err != nil {
		return err
	}

	// Create call and send request
	rpcCall := call{callId: 0, procedure: rpc, request: rpcRequest, response: rpcResponse}
	err = sendRequest(c, conn, &rpcCall)
	if err != nil {
		klog.Errorf("sendRequest error: %v", err)
		return err
	}

	// Read & return response
	err = c.readResponse(conn, &rpcCall)

	return err
}

var connectionPool = struct {
	sync.RWMutex
	connections map[connection_id]*connection
}{connections: make(map[connection_id]*connection)}

func findUsableTokenForService(service string) (*hadoop_common.TokenProto, bool) {
	userTokens := security.GetCurrentUser().GetUserTokens()

	klog.V(5).Infof("looking for token for service: %s\n", service)

	if len(userTokens) == 0 {
		return nil, false
	}

	token := userTokens[service]
	if token != nil {
		return token, true
	}

	return nil, false
}

func getAuthProtocol(c *Client) yarnauth.AuthProtocol {
	authProtocol := yarnauth.AUTH_PROTOCOL_NONE

	// TODO: check if we have a token for the service
	if _, found := findUsableTokenForService(c.ServerAddress); found {
		klog.V(4).Infof("found token for service: %s", c.ServerAddress)
		authProtocol = yarnauth.AUTH_PROTOCOL_SASL
	}

	if c.UGI.IsSecurityEnabled() {
		authProtocol = yarnauth.AUTH_PROTOCOL_SASL
	}

	return authProtocol
}

func getConnection(c *Client, connectionId *connection_id) (conn *connection, err error) {
	// Try to re-use an existing connection
	connectionPool.RLock()
	conn = connectionPool.connections[*connectionId]
	connectionPool.RUnlock()

	if conn != nil {
		return conn, nil
	}

	// If necessary, create a new connection and save it in the connection-pool
	conn, err = setupConnection(c)
	if err != nil {
		klog.Errorf("couldn't setup connection: %v", err)
		return nil, err
	}

	connectionPool.Lock()
	connectionPool.connections[*connectionId] = conn
	connectionPool.Unlock()

	authProtocol := getAuthProtocol(c)
	err = writeConnectionHeader(conn, authProtocol)
	if err != nil {
		return nil, err
	}
	var authMethod yarnauth.AuthMethod
	if authProtocol == yarnauth.AUTH_PROTOCOL_SASL {
		authMethod, err = setupSaslConnection(c, conn)
		if err != nil {
			klog.Errorf("couldn't setup sasl connection: %v", err)
			return nil, err
		}
		klog.V(5).Infof("proceeding with sasl. auth method: %v", authMethod)
	}

	err = writeConnectionContext(c, conn, connectionId, authMethod)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func setupSaslConnection(c *Client, con *connection) (yarnauth.AuthMethod, error) {
	authMethod, err := saslConnect(c, con)
	return *authMethod, err
}

func setupConnection(c *Client) (*connection, error) {
	d := net.Dialer{Timeout: connDefaultTimeout}
	conn, err := d.Dial("tcp", c.ServerAddress)
	if err != nil {
		klog.V(4).Infof("error: %v", err)
		return nil, err
	}

	// close the connection if we fail to setup the connection
	defer func() {
		if err != nil {
			// nolint:errcheck
			conn.Close()
		}
	}()

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("net.TCPConn type assert failed")
	}

	// TODO: Ping thread

	// Set tcp no-delay
	err = tcpConn.SetNoDelay(c.TCPNoDelay)
	if err != nil {
		return nil, err
	}

	err = tcpConn.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	if c.TCPLowLatency {
		var fd *os.File
		if fd, err = tcpConn.File(); err != nil {
			return nil, err
		}
		// nolint:errcheck
		defer fd.Close()

		/*
			This allows intermediate switches to shape IPC traffic
			differently from Shuffle/HDFS DataStreamer traffic.
			IPTOS_RELIABILITY (0x04) | IPTOS_LOWDELAY (0x10)
			Prefer to optimize connect() speed & response latency over net
			throughput.
		*/
		sockFd := int(fd.Fd())
		if err = syscall.SetsockoptInt(sockFd, syscall.IPPROTO_IP, syscall.IP_TOS, 0x14); err != nil {
			return nil, err
		}
		// TODO socket.setPerformancePreferences(1, 2, 0)
	}

	return &connection{tcpConn}, nil
}

func saslConnect(client *Client, con *connection) (*yarnauth.AuthMethod, error) {
	authMethod := yarnauth.AUTH_SIMPLE

	// send a SASL negotiation request
	if err := sendSaslMessage(con, &saslNegotiateMessage); err != nil {
		klog.Warningf("failed to send SASL NEGOTIATE message!")
		return nil, err
	}

	var saslClient SASLClient
	done := false
	for !done {
		saslResponseMessage, err := receiveSaslMessage(con)
		if err != nil {
			klog.Error("failed to receive SASL NEGOTIATE response!")
			return nil, err
		}

		challengeToken := saslResponseMessage.GetToken()
		var rpcResponse *hadoop_common.RpcSaslProto

		switch saslResponseMessage.GetState() {
		case hadoop_common.RpcSaslProto_NEGOTIATE:
			auth, client, err := selectSaslClient(client, saslResponseMessage.GetAuths())
			if err != nil {
				klog.Error("failed to select SASL client!")
				return nil, err
			}
			saslClient = client
			authMethod = yarnauth.ConvertAuthProtoToAuthMethod(auth)

			var responseToken []byte
			if authMethod == yarnauth.AUTH_SIMPLE {
				done = true
			} else {
				if auth.GetChallenge() != nil {
					challengeToken = auth.GetChallenge()
					// TODO clean auth challenge
				} else {
					// TODO check saslClient.hasInitialResponse
					challengeToken = []byte{}
				}
				responseToken, err = saslClient.EvaluateChallenge(challengeToken)
				if err != nil {
					klog.Error("failed to evaluate challenge!")
					return nil, err
				}
			}
			rpcResponse = createSaslReply(hadoop_common.RpcSaslProto_INITIATE, responseToken)
			rpcResponse.Auths = append(rpcResponse.Auths, auth)
		case hadoop_common.RpcSaslProto_CHALLENGE:
			if saslClient == nil {
				return nil, errors.New("sasl client is nil")
			}
			responseToken, err := saslClient.EvaluateChallenge(challengeToken)
			if err != nil {
				klog.Error("failed to evaluate challenge!")
				return nil, err
			}
			rpcResponse = createSaslReply(hadoop_common.RpcSaslProto_RESPONSE, responseToken)
		case hadoop_common.RpcSaslProto_SUCCESS:
			if saslClient == nil {
				authMethod = yarnauth.AUTH_SIMPLE
			} else {
				// nolint:errcheck
				saslClient.EvaluateChallenge(challengeToken)
			}
			done = true
		}

		if rpcResponse != nil {
			if err = sendSaslMessage(con, rpcResponse); err != nil {
				klog.Error("failed to send SASL message!")
				return nil, err
			}
		}
	}

	return &authMethod, nil
}

func writeConnectionHeader(conn *connection, authProtocol yarnauth.AuthProtocol) error {
	if err := conn.con.SetDeadline(time.Now().Add(rwDefaultTimeout)); err != nil {
		return err
	}
	// RPC_HEADER
	if _, err := conn.con.Write(yarnauth.RPC_HEADER); err != nil {
		klog.Warningf("conn.Write yarnauth.RPC_HEADER %v", err)
		return err
	}

	// RPC_VERSION
	if _, err := conn.con.Write(yarnauth.VERSION); err != nil {
		klog.Warningf("conn.Write yarnauth.VERSION %v", err)
		return err
	}

	// RPC_SERVICE_CLASS
	if serviceClass, err := yarnauth.ConvertFixedToBytes(yarnauth.RPC_SERVICE_CLASS); err != nil {
		klog.Warningf("binary.Write %v", err)
		return err
	} else if _, err := conn.con.Write(serviceClass); err != nil {
		klog.Warningf("conn.Write RPC_SERVICE_CLASS %v", err)
		return err
	}

	// AuthProtocol
	if authProtocolBytes, err := yarnauth.ConvertFixedToBytes(authProtocol); err != nil {
		klog.Warningf("WTF AUTH_PROTOCOL %v", err)
		return err
	} else if _, err := conn.con.Write(authProtocolBytes); err != nil {
		klog.Warningf("conn.Write yarnauth.AUTH_PROTOCOL %v", err)
		return err
	}

	return nil
}

func writeConnectionContext(c *Client, conn *connection, connectionId *connection_id, authMethod yarnauth.AuthMethod) error {
	if err := conn.con.SetDeadline(time.Now().Add(rwDefaultTimeout)); err != nil {
		return err
	}
	// TODO fix me
	// ugiProto := &hadoop_common.UserInformationProto{}
	// if c.UGI != nil {
	// 	if authMethod == yarnauth.AUTH_KERBEROS {
	// 		// Real user was established as part of the connection.
	// 		// Send effective user only.
	// 		effectiveUser := c.UGI.GetEffectiveUser()
	// 		ugiProto.EffectiveUser = &effectiveUser
	// 	} else if authMethod == yarnauth.AUTH_TOKEN {
	// 		// With token, the connection itself establishes
	// 		// both real and effective user. Hence send none in header
	// 	} else { // Simple authentication
	// 		// No user info is established as part of the connection.
	// 		// Send both effective user and real user
	// 		effectiveUser := c.UGI.GetEffectiveUser()
	// 		ugiProto.EffectiveUser = &effectiveUser
	// 		if realUser := c.UGI.GetRealUser(); realUser != "" {
	// 			ugiProto.RealUser = &realUser
	// 		}
	// 	}
	// }
	ugiProto, _ := yarnauth.CreateSimpleUGIProto()
	ipcCtxProto := hadoop_common.IpcConnectionContextProto{Protocol: &connectionId.protocol}
	ipcCtxProto.UserInfo = ugiProto

	// Create RpcRequestHeaderProto
	var clientId [16]byte = [16]byte(*c.ClientId)

	rpcReqHeaderProto := hadoop_common.RpcRequestHeaderProto{
		RpcKind:    &yarnauth.RPC_PROTOCOL_BUFFFER,
		RpcOp:      &yarnauth.RPC_FINAL_PACKET,
		CallId:     &CONNECTION_CONTEXT_CALL_ID,
		ClientId:   clientId[0:16],
		RetryCount: &yarnauth.RPC_DEFAULT_RETRY_COUNT,
	}

	rpcReqHeaderProtoBytes, err := proto.Marshal(&rpcReqHeaderProto)
	if err != nil {
		klog.Warningf("proto.Marshal(&rpcReqHeaderProto) %v", err)
		return err
	}

	ipcCtxProtoBytes, err := proto.Marshal(&ipcCtxProto)
	if err != nil {
		klog.Warningf("proto.Marshal(&ipcCtxProto) %v", err)
		return err
	}

	totalLength := len(rpcReqHeaderProtoBytes) + sizeVarint(len(rpcReqHeaderProtoBytes)) + len(ipcCtxProtoBytes) + sizeVarint(len(ipcCtxProtoBytes))
	var tLen int32 = int32(totalLength)
	totalLengthBytes, err := yarnauth.ConvertFixedToBytes(tLen)

	if err != nil {
		klog.Warningf("ConvertFixedToBytes(totalLength) %v", err)
		return err
	} else if _, err := conn.con.Write(totalLengthBytes); err != nil {
		klog.Warningf("conn.con.Write(totalLengthBytes) %v", err)
		return err
	}

	if err := writeDelimitedBytes(conn, rpcReqHeaderProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, rpcReqHeaderProtoBytes) %v", err)
		return err
	}
	if err := writeDelimitedBytes(conn, ipcCtxProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, ipcCtxProtoBytes) %v", err)
		return err
	}

	return nil
}

func sizeVarint(x int) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}

func sendRequest(c *Client, conn *connection, rpcCall *call) error {
	klog.V(5).Infof("About to call RPC: %v", rpcCall.procedure)

	if err := conn.con.SetDeadline(time.Now().Add(rwDefaultTimeout)); err != nil {
		return err
	}

	// 0. RpcRequestHeaderProto
	var clientId [16]byte = [16]byte(*c.ClientId)
	rpcReqHeaderProto := hadoop_common.RpcRequestHeaderProto{
		RpcKind:    &yarnauth.RPC_PROTOCOL_BUFFFER,
		RpcOp:      &yarnauth.RPC_FINAL_PACKET,
		CallId:     &rpcCall.callId,
		ClientId:   clientId[0:16],
		RetryCount: &rpcCall.retryCount,
	}
	rpcReqHeaderProtoBytes, err := proto.Marshal(&rpcReqHeaderProto)
	if err != nil {
		klog.Warningf("proto.Marshal(&rpcReqHeaderProto) %v", err)
		return err
	}

	// 1. RequestHeaderProto
	requestHeaderProto := rpcCall.procedure
	requestHeaderProtoBytes, err := proto.Marshal(requestHeaderProto)
	if err != nil {
		klog.Warningf("proto.Marshal(&requestHeaderProto) %v", err)
		return err
	}

	// 2. Param
	paramProto := rpcCall.request
	paramProtoBytes, err := proto.Marshal(paramProto)
	if err != nil {
		klog.Warningf("proto.Marshal(&paramProto) %v", err)
		return err
	}

	totalLength := len(rpcReqHeaderProtoBytes) + sizeVarint(len(rpcReqHeaderProtoBytes)) + len(requestHeaderProtoBytes) + sizeVarint(len(requestHeaderProtoBytes)) + len(paramProtoBytes) + sizeVarint(len(paramProtoBytes))
	var tLen int32 = int32(totalLength)
	if totalLengthBytes, err := yarnauth.ConvertFixedToBytes(tLen); err != nil {
		klog.Warningf("ConvertFixedToBytes(totalLength) %v", err)
		return err
	} else {
		if _, err := conn.con.Write(totalLengthBytes); err != nil {
			klog.Warningf("conn.con.Write(totalLengthBytes) %v", err)
			return err
		}
	}

	if err := writeDelimitedBytes(conn, rpcReqHeaderProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, rpcReqHeaderProtoBytes) %v", err)
		return err
	}
	if err := writeDelimitedBytes(conn, requestHeaderProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, requestHeaderProtoBytes) %v", err)
		return err
	}
	if err := writeDelimitedBytes(conn, paramProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, paramProtoBytes) %v", err)
		return err
	}

	klog.V(5).Infof("Succesfully sent request of length: %v", totalLength)

	return nil
}

//func writeDelimitedTo(conn *connection, msg proto.Message) error {
//	msgBytes, err := proto.Marshal(msg)
//	if err != nil {
//		klog.Warningf("proto.Marshal(msg)", err)
//		return err
//	}
//	return writeDelimitedBytes(conn, msgBytes)
//}

func writeDelimitedBytes(conn *connection, data []byte) error {
	if _, err := conn.con.Write(protowire.AppendVarint(nil, uint64(len(data)))); err != nil {
		klog.Warningf("conn.con.Write(proto.EncodeVarint(uint64(len(data)))) %v", err)
		return err
	}
	if _, err := conn.con.Write(data); err != nil {
		klog.Warningf("conn.con.Write(data) %v", err)
		return err
	}

	return nil
}

func (c *Client) readResponse(conn *connection, rpcCall *call) error {
	// Read first 4 bytes to get total-length
	var totalLength int32 = -1
	var totalLengthBytes [4]byte
	if _, err := conn.con.Read(totalLengthBytes[0:4]); err != nil {
		klog.Warningf("conn.con.Read(totalLengthBytes) %v", err)
		return err
	}

	if err := yarnauth.ConvertBytesToFixed(totalLengthBytes[0:4], &totalLength); err != nil {
		klog.Warningf("yarnauth.ConvertBytesToFixed(totalLengthBytes, &totalLength) %v", err)
		return err
	}

	var responseBytes = make([]byte, totalLength)
	reader := bufio.NewReader(conn.con)
	read, err := io.ReadFull(reader, responseBytes)
	if err != nil {
		klog.Warningf("io.ReadFull(reader, responseBytes), %v", err)
		return err
	}
	if int32(read) != totalLength {
		return fmt.Errorf("actural read length %v does not match the total length %v", read, totalLength)
	}

	// Parse RpcResponseHeaderProto
	rpcResponseHeaderProto := hadoop_common.RpcResponseHeaderProto{}
	off, err := readDelimited(responseBytes[0:totalLength], &rpcResponseHeaderProto)
	if err != nil {
		klog.Warningf("readDelimited(responseBytes, rpcResponseHeaderProto) %v", err)
		return err
	}
	klog.V(5).Infof("Received rpcResponseHeaderProto = %v", rpcResponseHeaderProto.String())

	err = c.checkRpcHeader(&rpcResponseHeaderProto)
	if err != nil {
		klog.Warningf("c.checkRpcHeader failed %v, rpc header %+v", err, &rpcResponseHeaderProto)
		return err
	}

	if *rpcResponseHeaderProto.Status == hadoop_common.RpcResponseHeaderProto_SUCCESS {
		// Parse RpcResponseWrapper
		_, err = readDelimited(responseBytes[off:], rpcCall.response)
	} else {
		klog.V(4).Infof("RPC failed with status: %v", rpcResponseHeaderProto.Status.String())
		errorDetails := [4]string{
			rpcResponseHeaderProto.Status.String(),
			"ServerDidNotSetExceptionClassName",
			"ServerDidNotSetErrorMsg",
			"ServerDidNotSetErrorDetail"}
		if rpcResponseHeaderProto.ExceptionClassName != nil {
			errorDetails[0] = *rpcResponseHeaderProto.ExceptionClassName
		}
		if rpcResponseHeaderProto.ErrorMsg != nil {
			errorDetails[1] = *rpcResponseHeaderProto.ErrorMsg
		}
		if rpcResponseHeaderProto.ErrorDetail != nil {
			errorDetails[2] = rpcResponseHeaderProto.ErrorDetail.String()
		}
		err = errors.New(strings.Join(errorDetails[:], ":"))
	}
	return err
}

func readDelimited(rawData []byte, msg proto.Message) (int, error) {
	headerLength, off := protowire.ConsumeVarint(rawData)
	if off == 0 {
		klog.Warningf("proto.DecodeVarint(rawData) returned zero")
		return -1, nil
	}
	b := rawData[off : off+int(headerLength)]
	err := proto.Unmarshal(b, msg)
	if err != nil {
		klog.Warningf("proto.Unmarshal(rawData[off:off+headerLength]) %v", err)
		return -1, err
	}

	return off + int(headerLength), nil
}

func (c *Client) checkRpcHeader(rpcResponseHeaderProto *hadoop_common.RpcResponseHeaderProto) error {
	var callClientId = [16]byte(*c.ClientId)
	var headerClientId = rpcResponseHeaderProto.ClientId
	if rpcResponseHeaderProto.ClientId != nil {
		if !bytes.Equal(callClientId[0:16], headerClientId[0:16]) {
			klog.Warningf("Incorrect clientId: %v", headerClientId)
			return errors.New("incorrect clientId")
		}
	}
	return nil
}

func sendSaslMessage(conn *connection, message *hadoop_common.RpcSaslProto) error {
	saslRpcHeaderProto := hadoop_common.RpcRequestHeaderProto{
		RpcKind:    &yarnauth.RPC_PROTOCOL_BUFFFER,
		RpcOp:      &yarnauth.RPC_FINAL_PACKET,
		CallId:     &SASL_RPC_CALL_ID,
		ClientId:   SASL_RPC_DUMMY_CLIENT_ID,
		RetryCount: &SASL_RPC_INVALID_RETRY_COUNT,
	}

	saslRpcHeaderProtoBytes, err := proto.Marshal(&saslRpcHeaderProto)

	if err != nil {
		klog.Warningf("proto.Marshal(&saslRpcHeaderProto) %v", err)
		return err
	}

	saslRpcMessageProtoBytes, err := proto.Marshal(message)

	if err != nil {
		klog.Warningf("proto.Marshal(saslMessage) %v", err)
		return err
	}

	totalLength := len(saslRpcHeaderProtoBytes) + sizeVarint(len(saslRpcHeaderProtoBytes)) + len(saslRpcMessageProtoBytes) + sizeVarint(len(saslRpcMessageProtoBytes))
	var tLen int32 = int32(totalLength)
	if err := conn.con.SetDeadline(time.Now().Add(rwDefaultTimeout)); err != nil {
		return err
	}
	if totalLengthBytes, err := yarnauth.ConvertFixedToBytes(tLen); err != nil {
		klog.Warningf("ConvertFixedToBytes(totalLength) %v", err)
		return err
	} else {
		if _, err := conn.con.Write(totalLengthBytes); err != nil {
			klog.Warningf("conn.con.Write(totalLengthBytes) %v", err)
			return err
		}
	}
	if err := writeDelimitedBytes(conn, saslRpcHeaderProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, saslRpcHeaderProtoBytes) %v", err)
		return err
	}
	if err := writeDelimitedBytes(conn, saslRpcMessageProtoBytes); err != nil {
		klog.Warningf("writeDelimitedBytes(conn, saslRpcMessageProtoBytes) %v", err)
		return err
	}

	return nil
}

func receiveSaslMessage(conn *connection) (*hadoop_common.RpcSaslProto, error) {
	// Read first 4 bytes to get total-length
	var totalLength int32 = -1
	var totalLengthBytes [4]byte

	if err := conn.con.SetDeadline(time.Now().Add(rwDefaultTimeout)); err != nil {
		return nil, err
	}

	if _, err := conn.con.Read(totalLengthBytes[0:4]); err != nil {
		klog.Warningf("conn.con.Read(totalLengthBytes) %v", err)
		return nil, err
	}
	if err := yarnauth.ConvertBytesToFixed(totalLengthBytes[0:4], &totalLength); err != nil {
		klog.Warningf("yarnauth.ConvertBytesToFixed(totalLengthBytes, &totalLength) %v", err)
		return nil, err
	}

	var responseBytes []byte = make([]byte, totalLength)

	if _, err := conn.con.Read(responseBytes); err != nil {
		klog.Warningf("conn.con.Read(totalLengthBytes) %v", err)
		return nil, err
	}

	// Parse RpcResponseHeaderProto
	rpcResponseHeaderProto := hadoop_common.RpcResponseHeaderProto{}
	off, err := readDelimited(responseBytes[0:totalLength], &rpcResponseHeaderProto)
	if err != nil {
		klog.Warningf("readDelimited(responseBytes, rpcResponseHeaderProto) %v", err)
		return nil, err
	}

	err = checkSaslRpcHeader(&rpcResponseHeaderProto)
	if err != nil {
		klog.Warningf("checkSaslRpcHeader failed %v", err)
		return nil, err
	}

	var saslRpcMessage hadoop_common.RpcSaslProto

	if *rpcResponseHeaderProto.Status == hadoop_common.RpcResponseHeaderProto_SUCCESS {
		// Parse RpcResponseWrapper
		if _, err = readDelimited(responseBytes[off:], &saslRpcMessage); err != nil {
			klog.Warningf("failed to read sasl response!")
			return nil, err
		} else {
			return &saslRpcMessage, nil
		}
	} else {
		klog.V(4).Infof("RPC failed with status: %v", rpcResponseHeaderProto.Status.String())
		errorDetails := [4]string{rpcResponseHeaderProto.Status.String(), "ServerDidNotSetExceptionClassName", "ServerDidNotSetErrorMsg", "ServerDidNotSetErrorDetail"}
		if rpcResponseHeaderProto.ExceptionClassName != nil {
			errorDetails[0] = *rpcResponseHeaderProto.ExceptionClassName
		}
		if rpcResponseHeaderProto.ErrorMsg != nil {
			errorDetails[1] = *rpcResponseHeaderProto.ErrorMsg
		}
		if rpcResponseHeaderProto.ErrorDetail != nil {
			errorDetails[2] = rpcResponseHeaderProto.ErrorDetail.String()
		}
		err = errors.New(strings.Join(errorDetails[:], ":"))
		return nil, err
	}
}

func checkSaslRpcHeader(rpcResponseHeaderProto *hadoop_common.RpcResponseHeaderProto) error {
	var headerClientId []byte = rpcResponseHeaderProto.ClientId
	if rpcResponseHeaderProto.ClientId != nil {
		if !bytes.Equal(SASL_RPC_DUMMY_CLIENT_ID, headerClientId) {
			klog.Warningf("Incorrect clientId: %v", headerClientId)
			return errors.New("incorrect clientId")
		}
	}
	return nil
}

func createSaslReply(state hadoop_common.RpcSaslProto_SaslState, token []byte) *hadoop_common.RpcSaslProto {
	return &hadoop_common.RpcSaslProto{
		State: &state,
		Token: token,
	}
}

// TODO implement this
func isValidAuthType(auth *hadoop_common.RpcSaslProto_SaslAuth) bool {
	return true
}

func selectSaslClient(client *Client, auths []*hadoop_common.RpcSaslProto_SaslAuth) (*hadoop_common.RpcSaslProto_SaslAuth, SASLClient, error) {
	var selectedAuth *hadoop_common.RpcSaslProto_SaslAuth
	var saslClient SASLClient
	var switchToSimple bool

	for _, auth := range auths {
		if !isValidAuthType(auth) {
			continue
		}

		authMethod := yarnauth.ConvertAuthProtoToAuthMethod(auth)
		if authMethod == yarnauth.AUTH_SIMPLE {
			switchToSimple = true
		} else {
			saslClient = createSaslClient(client, auth, authMethod)
			if saslClient == nil {
				continue
			}
		}
		selectedAuth = auth
		break
	}

	if saslClient == nil && !switchToSimple {
		return nil, nil, fmt.Errorf("client cannot authenticate via %v", auths)
	}
	return selectedAuth, saslClient, nil
}

func createSaslClient(c *Client, auth *hadoop_common.RpcSaslProto_SaslAuth, authMethod yarnauth.AuthMethod) SASLClient {
	switch authMethod {
	case yarnauth.AUTH_KERBEROS:
		keytabFilePath := c.UGI.GetKeytabFilePath()
		principal := c.UGI.GetPrincipal()
		return CreateKerberosClient(keytabFilePath, principal)
	case yarnauth.AUTH_TOKEN:
		token, found := findUsableTokenForService(c.ServerAddress)
		if !found {
			klog.Info("not found token")
			return nil
		}
		return CreateTokenAuth(token, auth)
	}

	return nil
}

type SASLClient interface {
	EvaluateChallenge([]byte) ([]byte, error)
}
