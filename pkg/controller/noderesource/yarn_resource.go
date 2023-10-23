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

// NOTE: functions in this file can be overwritten for extension

package noderesource

import (
	"github.com/koordinator-sh/koordinator/apis/extension"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	BatchCPU    = extension.BatchCPU
	BatchMemory = extension.BatchMemory

	YarnClusterIDAnnotation = "yarn.hadoop.apache.org/cluster-id"

	// TODO mv to koordinator/api
	yarnNodeAllocatedResourceAnnotationKey = "node.yarn.koordinator.sh/resourceAllocated"
	nodeOriginAllocatableAnnotationKey     = "node.koordinator.sh/originAllocatable"
)

const (
	yarnNodeCPUResource             = "yarn_node_cpu_resource"
	yarnNodeMemoryResource          = "yarn_node_memory_resource"
	yarnNodeCPUAllocatedResource    = "yarn_node_cpu_allocated_resource"
	yarnNodeMemoryAllocatedResource = "yarn_node_memory_allocated_resource"
)

func calculate(batchCPU resource.Quantity, batchMemory resource.Quantity) (int64, int64) {
	// TODO multiple ratio as buffer
	return batchCPU.ScaledValue(resource.Kilo), batchMemory.ScaledValue(resource.Mega)
}
