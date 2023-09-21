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

func GetAllKnownClusterID() ([]string, error) {
	dir := os.Getenv("HADOOP_CONF_DIR")
	res := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
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

func GetAllKnownClients() (map[string]*YarnClient, error) {
	ids, err := GetAllKnownClusterID()
	if err != nil {
		return nil, err
	}
	clients := map[string]*YarnClient{}
	for _, id := range ids {
		yClient, err := CreateYarnClientByClusterID(id)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		clients[id] = yClient
		klog.V(3).Infof("init yarn client %s", id)
	}
	return clients, nil
}
