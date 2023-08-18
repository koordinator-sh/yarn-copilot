/*
Copyright 2013 The Cloudera Inc.
Copyright 2023 The Koordinator Authors.

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

package conf

import (
	"fmt"
	"strings"
)

var (
	YARN_DEFAULT Resource = Resource{"yarn-default.xml", false}
	YARN_SITE    Resource = Resource{"yarn-site.xml", true}
)

const (
	YARN_PREFIX              = "yarn."
	RM_PREFIX                = YARN_PREFIX + "resourcemanager."
	RM_ADDRESS               = RM_PREFIX + "address"
	RM_SCHEDULER_ADDRESS     = RM_PREFIX + "scheduler.address"
	RM_ADMIN_ADDRESS         = RM_PREFIX + "admin.address"
	RM_HA_ENABLED            = RM_PREFIX + "ha.enabled"
	RM_HA_RM_IDS             = RM_PREFIX + "ha.rm-ids"
	RM_AM_EXPIRY_INTERVAL_MS = YARN_PREFIX + "am.liveness-monitor.expiry-interval-ms"

	DEFAULT_RM_ADDRESS               = "0.0.0.0:8032"
	DEFAULT_RM_SCHEDULER_ADDRESS     = "0.0.0.0:8030"
	DEFAULT_RM_ADMIN_ADDRESS         = "0.0.0.0:8033"
	DEFAULT_RM_AM_EXPIRY_INTERVAL_MS = 600000
	DEFAULT_RM_HA_ENABLED            = false
)

type yarn_configuration struct {
	conf Configuration
}

type YarnConfiguration interface {
	GetRMAddress() (string, error)
	GetRMSchedulerAddress() (string, error)
	GetRMAdminAddress() (string, error)
	GetRMEnabledHA() (bool, error)
	GetRMs() ([]string, error)
	GetRMAdminAddressByID(rmID string) (string, error)
	GetRMAddressByID(rmID string) (string, error)

	SetRMAddress(address string) error
	SetRMSchedulerAddress(address string) error

	Get(key string, defaultValue string) (string, error)
	GetInt(key string, defaultValue int) (int, error)

	Set(key string, value string) error
	SetInt(key string, value int) error
}

func (yarnConf *yarn_configuration) Get(key string, defaultValue string) (string, error) {
	return yarnConf.conf.Get(key, defaultValue)
}

func (yarnConf *yarn_configuration) GetInt(key string, defaultValue int) (int, error) {
	return yarnConf.conf.GetInt(key, defaultValue)
}

func (yarnConf *yarn_configuration) GetRMAddress() (string, error) {
	return yarnConf.conf.Get(RM_ADDRESS, DEFAULT_RM_ADDRESS)
}

func (yarnConf *yarn_configuration) GetRMSchedulerAddress() (string, error) {
	return yarnConf.conf.Get(RM_SCHEDULER_ADDRESS, DEFAULT_RM_SCHEDULER_ADDRESS)
}

func (yarnConf *yarn_configuration) GetRMAdminAddress() (string, error) {
	return yarnConf.conf.Get(RM_ADMIN_ADDRESS, DEFAULT_RM_ADMIN_ADDRESS)
}

func (yarnConf *yarn_configuration) GetRMEnabledHA() (bool, error) {
	return yarnConf.conf.GetBool(RM_HA_ENABLED, DEFAULT_RM_HA_ENABLED)
}

func (yarnConf *yarn_configuration) GetRMs() ([]string, error) {
	rmIDs := make([]string, 0)
	allRMs, err := yarnConf.conf.Get(RM_HA_RM_IDS, "")
	if err != nil {
		return rmIDs, nil
	}
	rmIDs = strings.Split(allRMs, ",")
	return rmIDs, nil
}

func (yarnConf *yarn_configuration) GetRMAdminAddressByID(rmID string) (string, error) {
	// yarn.resourcemanager.admin.address.rm1
	rmAddrKey := fmt.Sprintf("%v.%v", RM_ADMIN_ADDRESS, rmID)
	return yarnConf.conf.Get(rmAddrKey, DEFAULT_RM_ADMIN_ADDRESS)
}

func (yarnConf *yarn_configuration) GetRMAddressByID(rmID string) (string, error) {
	// yarn.resourcemanager.address.rm1
	rmAddrKey := fmt.Sprintf("%v.%v", RM_ADDRESS, rmID)
	return yarnConf.conf.Get(rmAddrKey, DEFAULT_RM_ADDRESS)
}

func (yarnConf *yarn_configuration) Set(key string, value string) error {
	return yarnConf.conf.Set(key, value)
}

func (yarnConf *yarn_configuration) SetInt(key string, value int) error {
	return yarnConf.conf.SetInt(key, value)
}

func (yarnConf *yarn_configuration) SetRMAddress(address string) error {
	return yarnConf.conf.Set(RM_ADDRESS, address)
}

func (yarnConf *yarn_configuration) SetRMSchedulerAddress(address string) error {
	return yarnConf.conf.Set(RM_SCHEDULER_ADDRESS, address)
}

func NewYarnConfiguration(hadooConfDir string, clusterID string) (YarnConfiguration, error) {
	// for yarn-site.xml with cluster id, read from clusterid.yarn-site.xml
	c, err := NewConfigurationResources(hadooConfDir, []Resource{YARN_DEFAULT, YARN_SITE}, configPrefix(clusterID))
	return &yarn_configuration{conf: c}, err
}

func configPrefix(clusterID string) string {
	if clusterID != "" {
		return clusterID + "."
	}
	return ""
}
