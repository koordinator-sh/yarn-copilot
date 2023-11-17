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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/cache"
)

var (
	yarnNodeCPUMetric = prometheus.NewDesc(
		yarnNodeCPUResource,
		"yarn node cpu resource",
		[]string{"instance", "cluster"},
		nil)
	yarnNodeMemoryMetric = prometheus.NewDesc(
		yarnNodeMemoryResource,
		"yarn node memory resource",
		[]string{"instance", "cluster"},
		nil)
	yarnNodeCPUAllocatedMetric = prometheus.NewDesc(
		yarnNodeCPUAllocatedResource,
		"yarn node cpu resource",
		[]string{"instance", "cluster"},
		nil)
	yarnNodeMemoryAllocatedMetric = prometheus.NewDesc(
		yarnNodeMemoryAllocatedResource,
		"yarn node memory resource",
		[]string{"instance", "cluster"},
		nil)
)

type YarnMetricCollector struct {
	cache *cache.NodesSyncer
}

func NewYarnMetricCollector(cache *cache.NodesSyncer) *YarnMetricCollector {
	return &YarnMetricCollector{cache: cache}
}

func (y *YarnMetricCollector) Describe(descs chan<- *prometheus.Desc) {
	descs <- yarnNodeCPUMetric
	descs <- yarnNodeMemoryMetric
	descs <- yarnNodeCPUAllocatedMetric
	descs <- yarnNodeMemoryAllocatedMetric
}

func (y *YarnMetricCollector) Collect(metrics chan<- prometheus.Metric) {
	for clusterID, nodes := range y.cache.GetYarnNodeInfo() {
		for _, node := range nodes {
			metrics <- prometheus.MustNewConstMetric(
				yarnNodeCPUMetric,
				prometheus.GaugeValue,
				float64(node.Capability.GetVirtualCores()),
				node.NodeId.GetHost(),
				clusterID,
			)
			metrics <- prometheus.MustNewConstMetric(
				yarnNodeMemoryMetric,
				prometheus.GaugeValue,
				float64(node.Capability.GetMemory()*1024*1024),
				node.NodeId.GetHost(),
				clusterID,
			)
			metrics <- prometheus.MustNewConstMetric(
				yarnNodeCPUAllocatedMetric,
				prometheus.GaugeValue,
				float64(node.Used.GetVirtualCores()),
				node.NodeId.GetHost(),
				clusterID,
			)
			metrics <- prometheus.MustNewConstMetric(
				yarnNodeMemoryAllocatedMetric,
				prometheus.GaugeValue,
				float64(node.Used.GetMemory()*1024*1024),
				node.NodeId.GetHost(),
				clusterID,
			)
		}
	}
}
