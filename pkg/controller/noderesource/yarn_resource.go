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
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/koordinator-sh/koordinator/apis/extension"
)

const (
	BatchCPU    = extension.BatchCPU
	BatchMemory = extension.BatchMemory

	YarnClusterIDAnnotation = "yarn.hadoop.apache.org/cluster-id"

	// TODO mv to koordinator/api
	NodeOriginAllocatableAnnotationKey = "node.koordinator.sh/originAllocatable"
	NodeAllocatedResourceAnnotationKey = "node.koordinator.sh/resourceAllocated"
)

func calculate(batchCPU resource.Quantity, batchMemory resource.Quantity) (int64, int64) {
	// TODO multiple ratio as buffer
	return batchCPU.ScaledValue(resource.Kilo), batchMemory.ScaledValue(resource.Mega)
}

// TODO mv to koordiantor api
type OriginAllocatable struct {
	Resources corev1.ResourceList `json:"resources,omitempty"`
}

func GetOriginExtendAllocatable(annotations map[string]string) (*OriginAllocatable, error) {
	originAllocatableStr, exist := annotations[NodeOriginAllocatableAnnotationKey]
	if !exist {
		return nil, nil
	}
	originAllocatable := &OriginAllocatable{}
	if err := json.Unmarshal([]byte(originAllocatableStr), originAllocatable); err != nil {
		return nil, err
	}
	return originAllocatable, nil
}

type NodeAllocated struct {
	YARNAllocated corev1.ResourceList `json:"yarnAllocated,omitempty"`
}

func GetNodeAllocated(annotations map[string]string) (*NodeAllocated, error) {
	nodeAllocatedStr, exist := annotations[NodeAllocatedResourceAnnotationKey]
	if !exist {
		return nil, nil
	}
	nodeAllocated := &NodeAllocated{}
	if err := json.Unmarshal([]byte(nodeAllocatedStr), nodeAllocated); err != nil {
		return nil, err
	}
	return nodeAllocated, nil
}
