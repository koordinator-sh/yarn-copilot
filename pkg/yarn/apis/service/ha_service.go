package service

import (
	"encoding/json"
	"math"

	uuid "github.com/nu7hatch/gouuid"
	"google.golang.org/protobuf/proto"

	gohadoop "github.com/koordinator-sh/goyarn/pkg/yarn/apis/auth"
	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopcommon"
	hadoop_ipc_client "github.com/koordinator-sh/goyarn/pkg/yarn/client/ipc"
)

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type HAServiceProtocolService interface {
	GetServiceStatus(in *hadoopcommon.GetServiceStatusRequestProto, out *hadoopcommon.GetServiceStatusResponseProto) error
}

var HA_SERVICE_PROTOCOL = "org.apache.hadoop.ha.HAServiceProtocol"

type HAServiceProtocolServiceClient struct {
	*hadoop_ipc_client.Client
}

func (c *HAServiceProtocolServiceClient) GetServiceStatus(in *hadoopcommon.GetServiceStatusRequestProto, out *hadoopcommon.GetServiceStatusResponseProto) error {
	return c.Call(gohadoop.GetCalleeRPCRequestHeaderProto(&HA_SERVICE_PROTOCOL), in, out)
}

func DialHAServiceProtocolService(serverAddress string) (HAServiceProtocolService, error) {
	clientId, _ := uuid.NewV4()
	ugi, _ := gohadoop.CreateSimpleUGIProto()
	c := &hadoop_ipc_client.Client{ClientId: clientId, Ugi: ugi, ServerAddress: serverAddress}
	return &HAServiceProtocolServiceClient{c}, nil
}
