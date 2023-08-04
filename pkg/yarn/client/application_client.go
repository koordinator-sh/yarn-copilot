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
	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	yarnservice "github.com/koordinator-sh/goyarn/pkg/yarn/apis/service"
	yarnconf "github.com/koordinator-sh/goyarn/pkg/yarn/config"
)

type YarnApplicationClient struct {
	client yarnservice.ApplicationClientProtocolService
}

func CreateYarnApplicationClient(conf yarnconf.YarnConfiguration, rmAddress *string) (*YarnApplicationClient, error) {
	c, err := yarnservice.DialApplicationClientProtocolService(conf, rmAddress)
	return &YarnApplicationClient{client: c}, err
}

func (c *YarnApplicationClient) GetClusterNode(request *hadoopyarn.GetClusterNodesRequestProto) (*hadoopyarn.GetClusterNodesResponseProto, error) {
	response := &hadoopyarn.GetClusterNodesResponseProto{}
	err := c.client.GetClusterNodes(request, response)
	if err != nil {
		return response, err
	}
	return response, nil
}
