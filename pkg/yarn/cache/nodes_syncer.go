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

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	yarnclient "github.com/koordinator-sh/goyarn/pkg/yarn/client"
)

const (
	syncInterval = time.Second
)

// YARN RM only supports get all nodes from cluster, sync to cache for efficiency
type NodesSyncer struct {
	yarnClients map[string]yarnclient.YarnClient

	// <ClusterID, <NodeID, NodeInfo>>
	cache map[string]map[string]*hadoopyarn.NodeReportProto
	mtx   sync.RWMutex
}

func NewNodesSyncer(yarnClients map[string]yarnclient.YarnClient) *NodesSyncer {
	return &NodesSyncer{
		yarnClients: yarnClients,
		cache:       map[string]map[string]*hadoopyarn.NodeReportProto{},
		mtx:         sync.RWMutex{},
	}
}

func (r *NodesSyncer) GetNodeResource(yarnNode *YarnNode) (*hadoopyarn.NodeReportProto, bool) {
	if yarnNode == nil {
		return nil, false
	}
	key := r.getKey(yarnNode.Name, yarnNode.Port)
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	clusterCache, exist := r.cache[yarnNode.ClusterID]
	if !exist {
		return nil, false
	}
	data, exist := clusterCache[key]
	return data, exist
}

func (r *NodesSyncer) getKey(yarnNodeName string, yarnNodePort int32) string {
	return fmt.Sprintf("%s-%d", yarnNodeName, yarnNodePort)
}

func (r *NodesSyncer) Start(ctx context.Context) error {
	t := time.NewTicker(syncInterval)
	debug := time.NewTicker(syncInterval * 10)
	go func() {
		for {
			select {
			case <-t.C:
				if err := r.syncYARNNodeAllocatedResource(); err != nil {
					klog.Errorf("sync yarn node allocated resource failed, error: %v", err)
				}
			case <-debug.C:
				r.debug()
			case <-ctx.Done():
				klog.V(1).Infof("stop node syncer")
				return
			}
		}
	}()
	return nil
}

func (r *NodesSyncer) debug() {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	for clusterID, clusterCache := range r.cache {
		for key, value := range clusterCache {
			klog.V(3).Infof("debug cache: %s %s %d %d %d %d", clusterID, key, *value.Used.VirtualCores,
				*value.Used.Memory, *value.Capability.VirtualCores, *value.Capability.Memory)
		}
	}
}

// GetYarnNodeInfo get yarn node info from cache, read only result
// Warning: Do not edit any field of results
func (r *NodesSyncer) GetYarnNodeInfo() map[string][]*hadoopyarn.NodeReportProto {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	res := map[string][]*hadoopyarn.NodeReportProto{}
	for clusterID, clusterCache := range r.cache {
		var data []*hadoopyarn.NodeReportProto
		for _, proto := range clusterCache {
			data = append(data, proto)
		}
		res[clusterID] = data
	}
	return res
}

func (r *NodesSyncer) syncYARNNodeAllocatedResource() error {
	req := hadoopyarn.GetClusterNodesRequestProto{NodeStates: []hadoopyarn.NodeStateProto{hadoopyarn.NodeStateProto_NS_RUNNING}}
	res := map[string]map[string]*hadoopyarn.NodeReportProto{}
	for id, yarnClient := range r.yarnClients {
		nodes, err := yarnClient.GetClusterNodes(&req)
		if err != nil {
			initErr := yarnClient.Reinitialize()
			return fmt.Errorf("GetClusterNodes error %v, reinitialize error %v", err, initErr)
		}
		clusterCache := map[string]*hadoopyarn.NodeReportProto{}
		for _, reportProto := range nodes.GetNodeReports() {
			if reportProto.NodeId.Host == nil || reportProto.NodeId.Port == nil {
				klog.Warningf("got nil node from rm %v", id)
				continue
			}
			key := r.getKey(*reportProto.NodeId.Host, *reportProto.NodeId.Port)
			clusterCache[key] = reportProto
		}
		res[id] = clusterCache
	}
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.cache = res
	return nil
}
