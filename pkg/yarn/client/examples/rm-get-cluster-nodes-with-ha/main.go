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

package main

import (
	"log"

	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	yarnclient "github.com/koordinator-sh/goyarn/pkg/yarn/client"
)

func main() {
	// Create YarnClient
	yarnClient, _ := yarnclient.CreateYarnClient()
	yarnClient.Initialize()

	request := &hadoopyarn.GetClusterNodesRequestProto{
		NodeStates: []hadoopyarn.NodeStateProto{},
	}
	response, err := yarnClient.GetClusterNodes(request)

	if err != nil {
		log.Printf("GetClusterNode size %v, response %v", len(response.NodeReports), response)
		log.Fatal("GetClusterNode Error", err)
	}

	log.Printf("GetClusterNode response %v", response)
	log.Printf("GetClusterNode response length %v", len(response.NodeReports))
}
