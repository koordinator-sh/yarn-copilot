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

	gohadoop "github.com/koordinator-sh/goyarn/pkg/yarn/apis/auth"
	yarnserver "github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn/server"
	hadoop_ipc_client "github.com/koordinator-sh/goyarn/pkg/yarn/client/ipc"
	yarn_conf "github.com/koordinator-sh/goyarn/pkg/yarn/config"
)

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

var RESOURCE_MANAGER_ADMIN_PROTOCOL = "org.apache.hadoop.yarn.server.api.ResourceManagerAdministrationProtocolPB"

func init() {
}

type ResourceManagerAdministrationProtocolService interface {
	UpdateNodeResource(in *yarnserver.UpdateNodeResourceRequestProto, out *yarnserver.UpdateNodeResourceResponseProto) error
}

type ResourceManagerAdministrationProtocolServiceClient struct {
	*hadoop_ipc_client.Client
}

func (c *ResourceManagerAdministrationProtocolServiceClient) UpdateNodeResource(in *yarnserver.UpdateNodeResourceRequestProto, out *yarnserver.UpdateNodeResourceResponseProto) error {
	return c.Call(gohadoop.GetCalleeRPCRequestHeaderProto(&RESOURCE_MANAGER_ADMIN_PROTOCOL), in, out)
}

func DialResourceManagerAdministrationProtocolService(conf yarn_conf.YarnConfiguration, rmAddress *string) (ResourceManagerAdministrationProtocolService, error) {
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
	} else if serverAddress, err = conf.GetRMAdminAddress(); err != nil {
		return nil, err
	}

	c := &hadoop_ipc_client.Client{ClientId: clientId, Ugi: ugi, ServerAddress: serverAddress}
	return &ResourceManagerAdministrationProtocolServiceClient{c}, nil
}
