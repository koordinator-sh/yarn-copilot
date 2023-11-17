/*
Copyright 2022 The Koordinator Authors.

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

package service

import (
	"encoding/json"
	"math"

	uuid "github.com/nu7hatch/gouuid"
	"google.golang.org/protobuf/proto"

	gohadoop "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/auth"
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopyarn"
	hadoop_ipc_client "github.com/koordinator-sh/yarn-copilot/pkg/yarn/client/ipc"
	yarn_conf "github.com/koordinator-sh/yarn-copilot/pkg/yarn/config"
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
