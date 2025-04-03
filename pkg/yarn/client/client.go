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

	"k8s.io/klog/v2"

	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopcommon"
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopyarn"
	yarnserver "github.com/koordinator-sh/yarn-copilot/pkg/yarn/apis/proto/hadoopyarn/server"
	yarnconf "github.com/koordinator-sh/yarn-copilot/pkg/yarn/config"
)

type YarnClient interface {
	Initialize() error
	Reinitialize() error
	Close()
	UpdateNodeResource(request *yarnserver.UpdateNodeResourceRequestProto) (*yarnserver.UpdateNodeResourceResponseProto, error)
	GetClusterNodes(request *hadoopyarn.GetClusterNodesRequestProto) (*hadoopyarn.GetClusterNodesResponseProto, error)
}

var _ YarnClient = &yarnClient{}

type yarnClient struct {
	confDir              string
	conf                 yarnconf.YarnConfiguration
	haEnabled            bool
	activeRMAdminAddress *string
	activeRMAddress      *string
	clusterID            string

	applicationClient *YarnApplicationClient
}

func NewYarnClient(confDir string, clusterID string) YarnClient {
	return &yarnClient{confDir: confDir, clusterID: clusterID}
}

func (c *yarnClient) Initialize() error {
	if conf, err := yarnconf.NewYarnConfiguration(c.confDir, c.clusterID); err == nil {
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

	// ha not enabled, use default conf
	if !c.haEnabled {
		if rmAdminAddr, err := c.conf.GetRMAdminAddress(); err == nil {
			c.activeRMAdminAddress = &rmAdminAddr
		} else {
			return err
		}
		if rmAddr, err := c.conf.GetRMAddress(); err == nil {
			c.activeRMAddress = &rmAddr
		} else {
			return err
		}
		return nil
	}

	// ha enabled, get active rm address by id
	var activeRMID string
	var err error
	if activeRMID, err = c.GetActiveRMID(); err != nil {
		return err
	}
	if rmAdminAddr, err := c.conf.GetRMAdminAddressByID(activeRMID); err == nil {
		c.activeRMAdminAddress = &rmAdminAddr
	} else {
		return err
	}
	if rmAddress, err := c.conf.GetRMAddressByID(activeRMID); err == nil {
		c.activeRMAddress = &rmAddress
	} else {
		return err
	}

	return nil
}

func (c *yarnClient) Close() {
	c.activeRMAdminAddress = nil
	c.activeRMAddress = nil

	// reset application client
	c.applicationClient = nil
}

func (c *yarnClient) Reinitialize() error {
	klog.V(5).Info("Reinitialize yarn client")

	c.Close()
	return c.Initialize()
}

func (c *yarnClient) UpdateNodeResource(request *yarnserver.UpdateNodeResourceRequestProto) (*yarnserver.UpdateNodeResourceResponseProto, error) {
	if c.activeRMAdminAddress == nil && c.haEnabled {
		if err := c.Initialize(); err != nil {
			return nil, err
		}
	}
	// TODO check response error code and retry auto
	return c.updateNodeResource(request)
}

func (c *yarnClient) GetClusterNodes(request *hadoopyarn.GetClusterNodesRequestProto) (*hadoopyarn.GetClusterNodesResponseProto, error) {
	if c.activeRMAdminAddress == nil && c.haEnabled {
		if err := c.Initialize(); err != nil {
			return nil, err
		}
	}
	// TODO check response error code and retry auto
	return c.getClusterNodes(request)
}

func (c *yarnClient) GetActiveRMID() (string, error) {
	rmIDs, err := c.conf.GetRMs()
	if err != nil {
		return "", err
	}
	for _, rmID := range rmIDs {
		rmAdminAddr, err := c.conf.GetRMAdminAddressByID(rmID)
		if err != nil {
			return "", err
		}
		haClient, err := CreateYarnHAClient(rmAdminAddr, c.conf)
		if err != nil {
			return "", fmt.Errorf("create yarn %v ha client for %v failed %v", rmID, rmAdminAddr, err)
		}
		resp, err := haClient.GetServiceStatus(&hadoopcommon.GetServiceStatusRequestProto{})
		if err != nil {
			klog.V(4).Infof("get %v service status for %v failed %v, try next rm", rmID, rmAdminAddr, err)
			continue
		}
		if resp.State != nil && *resp.State == hadoopcommon.HAServiceStateProto_ACTIVE {
			return rmID, nil
		}
	}
	return "", fmt.Errorf("active rm not found in %v", rmIDs)
}

func (c *yarnClient) updateNodeResource(request *yarnserver.UpdateNodeResourceRequestProto) (*yarnserver.UpdateNodeResourceResponseProto, error) {
	// TODO keep client alive instead of create every time
	adminClient, err := CreateYarnAdminClient(c.conf, c.activeRMAdminAddress)
	if err != nil {
		return nil, err
	}
	return adminClient.UpdateNodeResource(request)
}

func (c *yarnClient) getClusterNodes(request *hadoopyarn.GetClusterNodesRequestProto) (*hadoopyarn.GetClusterNodesResponseProto, error) {
	// reuse application client
	if c.applicationClient == nil {
		cl, err := CreateYarnApplicationClient(c.conf, c.activeRMAddress)
		if err != nil {
			return nil, err
		}

		c.applicationClient = cl
	}

	if c.applicationClient == nil {
		return nil, fmt.Errorf("null application client")
	}
	return c.applicationClient.GetClusterNode(request)
}
