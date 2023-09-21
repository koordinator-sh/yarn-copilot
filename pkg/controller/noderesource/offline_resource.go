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

package noderesource

import (
	"github.com/koordinator-sh/koordinator/apis/extension"
)

const (
	BatchCPU    = extension.BatchCPU
	BatchMemory = extension.BatchMemory

	YarnClusterIDAnnotation = "yarn.hadoop.apache.org/cluster-id"

	yarnNodeAllocatedResourceAnnotationKey = "node.yarn.koordinator.sh/resourceAllocated"
	actualOfflineResourceAnnotationKey     = "node.yarn.koordinator.sh/actualOfflineResource"
)

const (
	yarnNodeCPUResource    = "yarn_node_cpu_resource"
	yarnNodeMemoryResource = "yarn_node_memory_resource"
)
