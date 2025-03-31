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
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/security"
	hadoop_ipc_client "github.com/koordinator-sh/yarn-copilot/pkg/yarn/client/ipc"
	yarn_conf "github.com/koordinator-sh/yarn-copilot/pkg/yarn/config"
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

func DialHAServiceProtocolService(serverAddress string, conf yarn_conf.YarnConfiguration) (HAServiceProtocolService, error) {
	clientId, _ := uuid.NewV4()
	ugi, _ := security.CreateUserGroupInformation(conf)
	c := &hadoop_ipc_client.Client{
		ClientId:      clientId,
		UGI:           ugi,
		ServerAddress: serverAddress,
	}
	return &HAServiceProtocolServiceClient{c}, nil
}
