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

package client

import (
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	yarnservice "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/service"
	yarn_conf "github.com/koordinator-sh/yarn-copilot/pkg/yarn/config"
)

type YarnHAClient struct {
	client yarnservice.HAServiceProtocolService
}

func CreateYarnHAClient(rmAddress string, conf yarn_conf.YarnConfiguration) (*YarnHAClient, error) {
	c, err := yarnservice.DialHAServiceProtocolService(rmAddress, conf)
	return &YarnHAClient{client: c}, err
}

func (c *YarnHAClient) GetServiceStatus(request *hadoopcommon.GetServiceStatusRequestProto) (*hadoopcommon.GetServiceStatusResponseProto, error) {
	response := &hadoopcommon.GetServiceStatusResponseProto{}
	err := c.client.GetServiceStatus(request, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
