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

	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopyarn"
	yarnserver "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopyarn/server"
	yarnclient "github.com/koordinator-sh/yarn-copilot/pkg/yarn/client"
)

func main() {
	// Create yarnClient
	yarnClient, _ := yarnclient.DefaultYarnClientFactory.CreateDefaultYarnClient()

	host := "0.0.0.0"
	port := int32(8041)
	vCores := int32(101)
	memoryMB := int64(10240)
	request := &yarnserver.UpdateNodeResourceRequestProto{
		NodeResourceMap: []*hadoopyarn.NodeResourceMapProto{
			{
				NodeId: &hadoopyarn.NodeIdProto{
					Host: &host,
					Port: &port,
				},
				ResourceOption: &hadoopyarn.ResourceOptionProto{
					Resource: &hadoopyarn.ResourceProto{
						Memory:       &memoryMB,
						VirtualCores: &vCores,
					},
				},
			},
		},
	}
	response, err := yarnClient.UpdateNodeResource(request)

	if err != nil {
		log.Fatal("yarnClient.UpdateNodeResource ", err)
	}

	log.Printf("UpdateNodeResource response %v", response)
}
