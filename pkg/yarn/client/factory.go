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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

const (
	envHadoopConfDir = "HADOOP_CONF_DIR"

	DefaultClusterID = "__default_yarn_cluster__"
)

type YarnClientFactory interface {
	CreateDefaultYarnClient() (YarnClient, error)
	CreateYarnClientByClusterID(clusterID string) (YarnClient, error)
	CreateAllYarnClients() (map[string]YarnClient, error)
}

var DefaultYarnClientFactory YarnClientFactory = &yarnClientFactory{configDir: os.Getenv(envHadoopConfDir)}

type yarnClientFactory struct {
	configDir string
}

func (f *yarnClientFactory) CreateDefaultYarnClient() (YarnClient, error) {
	c := NewYarnClient(f.configDir, "")
	if err := c.Initialize(); err != nil {
		return nil, err
	}
	return c, nil
}

func (f *yarnClientFactory) CreateYarnClientByClusterID(clusterID string) (YarnClient, error) {
	c := NewYarnClient(f.configDir, clusterID)
	if err := c.Initialize(); err != nil {
		return nil, err
	}
	return c, nil
}

func (f *yarnClientFactory) CreateAllYarnClients() (map[string]YarnClient, error) {
	ids, err := f.getAllKnownClusterID()
	if err != nil {
		return nil, err
	}
	clients := map[string]YarnClient{}
	for _, id := range ids {
		yClient, err := f.CreateYarnClientByClusterID(id)
		if err != nil {
			klog.Errorf("create yarn client %v failed, error %v", id, err)
			return nil, err
		}
		clients[id] = yClient
		klog.V(3).Infof("init yarn client %v", id)
	}
	if defaultClient, err := f.CreateDefaultYarnClient(); err == nil {
		clients[DefaultClusterID] = defaultClient
		klog.V(3).Infof("init yarn client %v", defaultClient)
	} else {
		klog.Errorf("create yarn client %v failed, error %v", DefaultClusterID, err)
		return nil, err
	}
	return clients, nil
}

func (f *yarnClientFactory) getAllKnownClusterID() ([]string, error) {
	res := []string{}
	err := filepath.WalkDir(f.configDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".yarn-site.xml") {
			res = append(res, strings.ReplaceAll(d.Name(), ".yarn-site.xml", ""))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
