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

var (
	FinalContainerStates = []string{"KILLING", "DONE", "LOCALIZATION_FAILED", "CONTAINER_RESOURCES_CLEANINGUP",
		"CONTAINER_CLEANEDUP_AFTER_KILL", "EXITED_WITH_FAILURE", "EXITED_WITH_SUCCESS"}
)

type YarnContainer struct {
	Id                  string   `json:"id"`
	Appid               string   `json:"appid"`
	State               string   `json:"state"`
	ExitCode            int      `json:"exitCode"`
	Diagnostics         string   `json:"diagnostics"`
	User                string   `json:"user"`
	TotalMemoryNeededMB int      `json:"totalMemoryNeededMB"`
	TotalVCoresNeeded   int      `json:"totalVCoresNeeded"`
	ContainerLogsLink   string   `json:"containerLogsLink"`
	NodeId              string   `json:"nodeId"`
	MemUsed             float64  `json:"memUsed"`
	MemMaxed            float64  `json:"memMaxed"`
	CpuUsed             float64  `json:"cpuUsed"`
	CpuMaxed            float64  `json:"cpuMaxed"`
	ContainerLogFiles   []string `json:"containerLogFiles"`
}

func (c *YarnContainer) IsFinalState() bool {
	for _, state := range FinalContainerStates {
		if c.State == state {
			return true
		}
	}
	return false
}
