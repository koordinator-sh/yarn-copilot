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

package nm

import (
	"fmt"

	"github.com/koordinator-sh/koordinator/pkg/koordlet/statesinformer"
)

const (
	ComponentLabelKey             = "app.kubernetes.io/component"
	NodeManagerComponentLabelName = "node-manager"
)

type NMPodWatcher struct {
	kubeletstub statesinformer.KubeletStub
}

func NewNMPodWater(kubeletstub statesinformer.KubeletStub) *NMPodWatcher {
	return &NMPodWatcher{kubeletstub: kubeletstub}
}

func (n *NMPodWatcher) GetNMPodEndpoint() (string, bool, error) {
	pods, err := n.kubeletstub.GetAllPods()
	if err != nil {
		return "", false, err
	}
	for _, pod := range pods.Items {
		if pod.Labels[ComponentLabelKey] != NodeManagerComponentLabelName {
			continue
		}
		if pod.Spec.HostNetwork {
			return "localhost:8042", true, nil
		}
		return fmt.Sprintf("%s:8042", pod.Status.PodIP), true, nil
	}
	return "", false, nil
}
