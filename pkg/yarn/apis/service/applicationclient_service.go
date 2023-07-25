package service

import (
	"encoding/json"
	"math"

	uuid "github.com/nu7hatch/gouuid"
	"google.golang.org/protobuf/proto"

	gohadoop "github.com/koordinator-sh/goyarn/pkg/yarn/apis/auth"
	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	hadoop_ipc_client "github.com/koordinator-sh/goyarn/pkg/yarn/client/ipc"
	yarn_conf "github.com/koordinator-sh/goyarn/pkg/yarn/config"
)

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

var APPLICATION_CLIENT_PROTOCOL = "org.apache.hadoop.yarn.api.ApplicationClientProtocolPB"

func init() {

}

type ApplicationClientProtocolService interface {
	GetClusterNodes(in *hadoopyarn.GetClusterNodesRequestProto, out *hadoopyarn.GetClusterNodesResponseProto) error
}

var _ ApplicationClientProtocolService = &ApplicationClientProtocolServiceClient{}

type ApplicationClientProtocolServiceClient struct {
	*hadoop_ipc_client.Client
}

func (c *ApplicationClientProtocolServiceClient) GetClusterNodes(in *hadoopyarn.GetClusterNodesRequestProto, out *hadoopyarn.GetClusterNodesResponseProto) error {
	return c.Call(gohadoop.GetCalleeRPCRequestHeaderProto(&APPLICATION_CLIENT_PROTOCOL), in, out)
}

func DialApplicationClientProtocolService(conf yarn_conf.YarnConfiguration, rmAddress *string) (ApplicationClientProtocolService, error) {
	clientId, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	ugi, err := gohadoop.CreateSimpleUGIProto()
	if err != nil {
		return nil, err
	}

	var serverAddress string
	if rmAddress != nil {
		serverAddress = *rmAddress
	} else if serverAddress, err = conf.GetRMAddress(); err != nil {
		return nil, err
	}

	c := &hadoop_ipc_client.Client{ClientId: clientId, Ugi: ugi, ServerAddress: serverAddress}
	return &ApplicationClientProtocolServiceClient{c}, nil
}
