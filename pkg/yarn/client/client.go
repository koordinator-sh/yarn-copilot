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
	"fmt"
	"os"

	"k8s.io/klog/v2"

	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopcommon"
	yarnserver "github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn/server"
	yarnconf "github.com/koordinator-sh/goyarn/pkg/yarn/config"
)

type YARNClient struct {
	conf            yarnconf.YarnConfiguration
	haEnabled       bool
	activeRMAddress *string
	rmAddress       map[string]string
}

func CreateYARNClient() (*YARNClient, error) {
	c := &YARNClient{rmAddress: map[string]string{}}
	if err := c.Initialize(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *YARNClient) Initialize() error {
	if conf, err := yarnconf.NewYarnConfiguration(os.Getenv("HADOOP_CONF_DIR")); err == nil {
		// TODO use flags for conf dir config
		c.conf = conf
	} else {
		return err
	}

	if ha, err := c.conf.GetRMEnabledHA(); err == nil {
		c.haEnabled = ha
	} else {
		return err
	}

	if !c.haEnabled {
		if rmAddr, err := c.conf.GetRMAdminAddress(); err == nil {
			c.activeRMAddress = &rmAddr
		} else {
			return err
		}
		return nil
	}

	rmIDs, err := c.conf.GetRMs()
	if err != nil {
		return err
	}
	for _, rmID := range rmIDs {
		if rmAddr, err := c.conf.GetRMAddressByID(rmID); err == nil {
			c.rmAddress[rmID] = rmAddr
		} else {
			return err
		}
	}

	if rmAddr, err := c.GetActiveRMAddress(); err == nil {
		c.activeRMAddress = &rmAddr
	} else {
		return err
	}

	return nil
}

func (c *YARNClient) Close() {
	c.activeRMAddress = nil
}

func (c *YARNClient) Reinitialize() error {
	c.Close()
	return c.Initialize()
}

func (c *YARNClient) UpdateNodeResource(request *yarnserver.UpdateNodeResourceRequestProto) (*yarnserver.UpdateNodeResourceResponseProto, error) {
	if c.activeRMAddress == nil && c.haEnabled {
		if rmAddr, err := c.GetActiveRMAddress(); err != nil {
			return nil, err
		} else {
			c.activeRMAddress = &rmAddr
		}
	}
	// TODO check response error code and retry auto
	return c.updateNodeResource(request)
}

func (c *YARNClient) GetActiveRMAddress() (string, error) {
	for _, rmAddr := range c.rmAddress {
		haClient, err := CreateYarrHAClient(rmAddr)
		if err != nil {
			return "", fmt.Errorf("create yarn ha client for %v failed %v", rmAddr, err)
		}
		resp, err := haClient.GetServiceStatus(&hadoopcommon.GetServiceStatusRequestProto{})
		if err != nil {
			klog.V(4).Infof("get service status for %v failed %v, try next rm", rmAddr, err)
			continue
		}
		if resp.State != nil && *resp.State == hadoopcommon.HAServiceStateProto_ACTIVE {
			return rmAddr, nil
		}
	}
	return "", fmt.Errorf("active rm not found in %v", c.rmAddress)
}

func (c *YARNClient) updateNodeResource(request *yarnserver.UpdateNodeResourceRequestProto) (*yarnserver.UpdateNodeResourceResponseProto, error) {
	// TODO keep client alive instead of create every time
	adminClient, err := CreateYarnAdminClient(c.conf, c.activeRMAddress)
	if err != nil {
		return nil, err
	}
	return adminClient.UpdateNodeResource(request)
}
