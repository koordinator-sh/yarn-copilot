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
	"os"

	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	yarnclient "github.com/koordinator-sh/yarn-copilot/pkg/yarn/client"
	yarnconf "github.com/koordinator-sh/yarn-copilot/pkg/yarn/config"
)

func main() {
	// Create YarnConfiguration
	conf, err := yarnconf.NewYarnConfiguration(os.Getenv("HADOOP_CONF_DIR"), "")
	if err != nil {
		log.Fatal("new yarn conf", err)
	}

	rmIDs, err := conf.GetRMs()
	if err != nil {
		log.Fatal("get rms from conf", err)
	}

	for _, rmID := range rmIDs {
		rmAddr, err := conf.GetRMAdminAddressByID(rmID)
		if err != nil {
			log.Fatal("GetRMAdminAddressByID", err)
		}

		// Create YarnAdminClient
		yarnHAClient, _ := yarnclient.CreateYarnHAClient(rmAddr)

		request := &hadoopcommon.GetServiceStatusRequestProto{}
		response, err := yarnHAClient.GetServiceStatus(request)

		if err != nil {
			log.Fatal("yarnHAClient.GetServiceStatus ", err)
		}

		log.Printf("GetServiceStatus response %v", response)
	}
}
