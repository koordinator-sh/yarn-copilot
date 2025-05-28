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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/koordinator-sh/koordinator/apis/extension"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/koordinator-sh/yarn-copilot/pkg/copilot-agent/nm"
)

// MockNodeMangerOperator 模拟NodeMangerOperator接口
type MockNodeMangerOperator struct {
	containers *nm.Containers
	err1       error
	err2       error
}

func (m *MockNodeMangerOperator) Run(stop <-chan struct{}) error {
	return nil
}

func (m *MockNodeMangerOperator) KillContainer(containerID string) error {
	return m.err2
}

func (m *MockNodeMangerOperator) ListContainers() (*nm.Containers, error) {
	return m.containers, m.err1
}

func (m *MockNodeMangerOperator) GetContainer(containerID string) (*nm.YarnContainer, error) {
	if m.containers == nil {
		return nil, m.err1
	}
	for _, container := range m.containers.Containers.Items {
		if container.Id == containerID {
			return &container, nil
		}
	}
	return nil, m.err1
}

func (m *MockNodeMangerOperator) GenerateCgroupPath(containerID string) string {
	return "yarn/" + containerID
}

func (m *MockNodeMangerOperator) GenerateCgroupFullPath(cgroupSubSystem string) string {
	return "/sys/fs/cgroup/" + cgroupSubSystem + "/yarn"
}

func setupTestServer() (*YarnCopilotServer, *MockNodeMangerOperator) {
	mockMgr := &MockNodeMangerOperator{}
	server := NewYarnCopilotServer(mockMgr, "/tmp/test.sock")
	return server, mockMgr
}

func TestYarnCopilotServer_Health(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, _ := setupTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/health", nil)

	server.Health(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "\"ok\"", w.Body.String())
}

func TestYarnCopilotServer_Information(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, _ := setupTestServer()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/information", nil)

	server.Information(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var info PluginInfo
	err := json.Unmarshal(w.Body.Bytes(), &info)
	assert.NoError(t, err)
	assert.Equal(t, "yarn", info.Name)
	assert.Equal(t, "v1", info.Version)
}

func TestYarnCopilotServer_ListContainers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, mockMgr := setupTestServer()

	mockMgr.containers = &nm.Containers{
		Containers: struct {
			Items []nm.YarnContainer `json:"container"`
		}{
			Items: []nm.YarnContainer{
				{
					Id:    "container_e517_1746198264116_5225_01_000003",
					State: "RUNNING",
				},
				{
					Id:    "container_e517_1746198264116_5225_01_000004",
					State: "DONE",
				},
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/containers", nil)

	server.ListContainers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var containers []*ContainerInfo
	err := json.Unmarshal(w.Body.Bytes(), &containers)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers)) // 只返回非终态容器
	assert.Equal(t, "container_e517_1746198264116_5225_01_000003", containers[0].Name)

	// 测试错误情况
	mockMgr.err1 = assert.AnError
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/containers", nil)

	server.ListContainers(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestYarnCopilotServer_GetContainer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, mockMgr := setupTestServer()

	mockMgr.containers = &nm.Containers{
		Containers: struct {
			Items []nm.YarnContainer `json:"container"`
		}{
			Items: []nm.YarnContainer{
				{
					Id:    "container_e517_1746198264116_5225_01_000003",
					State: "RUNNING",
				},
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/container?containerID=container_e517_1746198264116_5225_01_000003", nil)

	server.GetContainer(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var container ContainerInfo
	err := json.Unmarshal(w.Body.Bytes(), &container)
	assert.NoError(t, err)
	assert.Equal(t, "container_e517_1746198264116_5225_01_000003", container.Name)

	// 测试容器不存在的情况
	mockMgr.err1 = errors.New("container Not Found")
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/v1/container?containerID=container_e517_1746198264116_5225_01_000004", nil)

	server.GetContainer(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestYarnCopilotServer_KillContainer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, mockMgr := setupTestServer()

	mockMgr.containers = &nm.Containers{
		Containers: struct {
			Items []nm.YarnContainer `json:"container"`
		}{
			Items: []nm.YarnContainer{
				{
					Id:    "container_e517_1746198264116_5225_01_000003",
					State: "RUNNING",
				},
			},
		},
	}

	killRequest := KillRequest{
		ContainerID: "container_e517_1746198264116_5225_01_000003",
	}
	body, _ := json.Marshal(killRequest)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/killContainer", bytes.NewBuffer(body))

	server.KillContainer(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var killInfo KillInfo
	err := json.Unmarshal(w.Body.Bytes(), &killInfo)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(killInfo.Items))
	assert.Equal(t, "container_e517_1746198264116_5225_01_000003", killInfo.Items[0].Name)

	// 测试错误情况
	mockMgr.err2 = assert.AnError
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/killContainer", bytes.NewBuffer(body))

	server.KillContainer(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestYarnCopilotServer_KillContainersByResource(t *testing.T) {
	var tests = []struct {
		name              string
		containers        []nm.YarnContainer
		killRequest       KillRequest
		releasedResources v1.ResourceList
		httpStatus        int
		err1              error
		err2              error
	}{

		{
			name: "list container error ",
			containers: []nm.YarnContainer{
				{
					Id:                  "container_e517_1746198264116_5225_01_000003",
					State:               "RUNNING",
					TotalVCoresNeeded:   2,
					TotalMemoryNeededMB: 1024,
				},
			},
			killRequest: KillRequest{
				Resources: v1.ResourceList{
					extension.BatchCPU:    resource.MustParse("1"),
					extension.BatchMemory: resource.MustParse("1Gi"),
				},
			},
			releasedResources: v1.ResourceList{},
			httpStatus:        http.StatusInternalServerError,
			err1:              assert.AnError,
			err2:              nil,
		},
		{
			name: "kill container error ",
			containers: []nm.YarnContainer{
				{
					Id:                  "container_e517_1746198264116_5225_01_000003",
					State:               "RUNNING",
					TotalVCoresNeeded:   2,
					TotalMemoryNeededMB: 1024,
				},
			},
			killRequest: KillRequest{
				Resources: v1.ResourceList{
					extension.BatchCPU:    resource.MustParse("1"),
					extension.BatchMemory: resource.MustParse("1Gi"),
				},
			},
			releasedResources: v1.ResourceList{},
			httpStatus:        http.StatusInternalServerError,
			err1:              nil,
			err2:              assert.AnError,
		},
		{
			name: "kill container with max containerID",
			containers: []nm.YarnContainer{
				{
					Id:                  "container_e517_1746198264116_5225_01_000003",
					State:               "RUNNING",
					TotalVCoresNeeded:   2,
					TotalMemoryNeededMB: 1024,
				},
				{
					Id:                  "container_e517_1746198264116_34108_01_000003",
					State:               "RUNNING",
					TotalVCoresNeeded:   1,
					TotalMemoryNeededMB: 2048,
				},
			},
			killRequest: KillRequest{
				Resources: v1.ResourceList{
					extension.BatchCPU:    resource.MustParse("1"),
					extension.BatchMemory: resource.MustParse("1Gi"),
				},
			},
			releasedResources: v1.ResourceList{
				extension.BatchCPU:    resource.MustParse("1"),
				extension.BatchMemory: resource.MustParse("2Gi"),
			},
			httpStatus: http.StatusOK,
			err1:       nil,
			err2:       nil,
		},
		{
			name: "kill container which is not ApplicationMaster",
			containers: []nm.YarnContainer{
				{
					Id:                  "container_e517_1746198264116_5225_01_000003",
					State:               "RUNNING",
					TotalVCoresNeeded:   2,
					TotalMemoryNeededMB: 1024,
				},
				{
					Id:                  "container_e517_1746198264116_34108_01_000001",
					State:               "RUNNING",
					TotalVCoresNeeded:   1,
					TotalMemoryNeededMB: 2048,
				},
			},
			killRequest: KillRequest{
				Resources: v1.ResourceList{
					extension.BatchCPU:    resource.MustParse("1"),
					extension.BatchMemory: resource.MustParse("1Gi"),
				},
			},
			releasedResources: v1.ResourceList{
				extension.BatchCPU:    resource.MustParse("2"),
				extension.BatchMemory: resource.MustParse("1Gi"),
			},
			httpStatus: http.StatusOK,
			err1:       nil,
			err2:       nil,
		},
		{
			name: "when all containers running with ApplicationMaster，kill container with max containerId",
			containers: []nm.YarnContainer{
				{
					Id:                  "container_e517_1746198264116_5225_01_000001",
					State:               "RUNNING",
					TotalVCoresNeeded:   2,
					TotalMemoryNeededMB: 1024,
				},
				{
					Id:                  "container_e517_1746198264116_34108_01_000001",
					State:               "RUNNING",
					TotalVCoresNeeded:   1,
					TotalMemoryNeededMB: 2048,
				},
			},
			killRequest: KillRequest{
				Resources: v1.ResourceList{
					extension.BatchCPU:    resource.MustParse("1"),
					extension.BatchMemory: resource.MustParse("1Gi"),
				},
			},
			releasedResources: v1.ResourceList{
				extension.BatchCPU:    resource.MustParse("1"),
				extension.BatchMemory: resource.MustParse("2Gi"),
			},
			httpStatus: http.StatusOK,
			err1:       nil,
			err2:       nil,
		},
	}
	for _, tt := range tests {
		gin.SetMode(gin.TestMode)
		server, mockMgr := setupTestServer()
		mockMgr.err1 = tt.err1
		mockMgr.err2 = tt.err2
		mockMgr.containers = &nm.Containers{
			Containers: struct {
				Items []nm.YarnContainer `json:"container"`
			}{
				Items: tt.containers,
			},
		}
		body, _ := json.Marshal(tt.killRequest)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/v1/killContainersByResource", bytes.NewBuffer(body))
		server.KillContainersByResource(c)
		assert.Equal(t, tt.httpStatus, w.Code)
		var releasedResources v1.ResourceList
		err := json.Unmarshal(w.Body.Bytes(), &releasedResources)
		assert.NoError(t, err)
		assert.NotNil(t, releasedResources)
		assert.Equal(t, tt.releasedResources, releasedResources)
	}
}

func TestYarnCopilotServer_Run(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, _ := setupTestServer()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := server.Run(ctx)
	assert.NoError(t, err)
}

func Test_filter(t *testing.T) {
	tests := []struct {
		name       string
		containers []nm.YarnContainer
		expected   int
	}{
		{
			name: "filter container which containerID ended with 000001",
			containers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_01_000001"},
				{Id: "container_e517_1746198264116_5225_01_000002"},
				{Id: "container_e517_1746198264116_34108_01_000003"},
			},
			expected: 2,
		},
		{
			name: "filter all containers",
			containers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_01_000001"},
				{Id: "container_e517_1746198264116_5226_01_000001"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter(tt.containers)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

func Test_score(t *testing.T) {
	tests := []struct {
		name               string
		containers         []nm.YarnContainer
		expectedContainers []nm.YarnContainer
	}{
		{
			name: "sort containerId",
			containers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_01_000003"},
				{Id: "container_e517_1746198264116_5225_01_000004"},
				{Id: "container_e517_1746198264116_34108_01_000003"},
				{Id: "container_e517_1746198264116_34108_01_000003"},
			},
			expectedContainers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_34108_01_000003"},
				{Id: "container_e517_1746198264116_34108_01_000003"},
				{Id: "container_e517_1746198264116_5225_01_000004"},
				{Id: "container_e517_1746198264116_5225_01_000003"},
			},
		},
		{
			// container_*clusterTimestamp*_*appId*_*attemptId*_*containerId*
			name: "sort attemptId",
			containers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_01_000004"},
				{Id: "container_e517_1746198264116_5225_02_000003"},
			},
			expectedContainers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_02_000003"},
				{Id: "container_e517_1746198264116_5225_01_000004"},
			},
		},
		{
			// container_*clusterTimestamp*_*appId*_*attemptId*_*containerId*
			name: "sort appId",
			containers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_02_000004"},
				{Id: "container_e517_1746198264116_5226_01_000003"},
			},
			expectedContainers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5226_01_000003"},
				{Id: "container_e517_1746198264116_5225_02_000004"},
			},
		},
		{
			// container_*clusterTimestamp*_*appId*_*attemptId*_*containerId*
			name: "sort clusterTimestamp",
			containers: []nm.YarnContainer{
				{Id: "container_e517_1746198264116_5225_02_000004"},
				{Id: "container_e517_1746198264117_5225_01_000003"},
			},
			expectedContainers: []nm.YarnContainer{
				{Id: "container_e517_1746198264117_5225_01_000003"},
				{Id: "container_e517_1746198264116_5225_02_000004"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := score(tt.containers)
			assert.Equal(t, tt.expectedContainers, result)
		})
	}
}

func Test_parseLastSegment(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		expected    string
		hasError    bool
	}{
		{
			name:        "valid containerID",
			containerID: "container_e517_1746198264116_5225_01_000001",
			expected:    "000001",
			hasError:    false,
		},
		{
			name:        "invalid containerID",
			containerID: "container_e517_1746198264116_5225_01_000003_000001",
			expected:    "",
			hasError:    true,
		},
		{
			name:        "empty containerID",
			containerID: "",
			expected:    "",
			hasError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLastSegment(tt.containerID)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_parseContainerID(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		expected    string
		hasError    bool
		expectedErr string
	}{
		{
			name:        "invalid container ID, too long",
			containerID: "container_e517_1746198264116_5225_01_000003_000001",
			hasError:    true,
			expectedErr: "invalid container ID format",
		},
		{
			name:        "invalid container ID, too short",
			containerID: "container_e517_1746198264116_5225",
			hasError:    true,
			expectedErr: "invalid container ID format",
		},
		{
			name:        "invalid container ID, not started with prefix container",
			containerID: "c_e517_1746198264116_5225_01",
			hasError:    true,
			expectedErr: "invalid container ID format",
		},
		{
			name:        "invalid cluster timestamp, number too long",
			containerID: "container_e517_17461982641167_5225_01_000001",
			hasError:    true,
			expectedErr: "invalid cluster timestamp",
		},
		{
			name:        "invalid cluster timestamp, not number",
			containerID: "container_e517_xxx_5225_01_000001",
			hasError:    true,
			expectedErr: "invalid cluster timestamp",
		},
		{
			name:        "invalid app ID, not number",
			containerID: "container_e517_1746198264116_xxx_01_000001",
			hasError:    true,
			expectedErr: "invalid app ID",
		},
		{
			name:        "invalid attempt ID, not number",
			containerID: "container_e517_1746198264116_5225_xx_000001",
			hasError:    true,
			expectedErr: "invalid attempt ID",
		},
		{
			name:        "invalid container ID, not number",
			containerID: "container_e517_1746198264116_5225_01_00000x",
			hasError:    true,
			expectedErr: "invalid container ID",
		},
		{
			name:        "valid container ID",
			containerID: "container_e517_1746198264116_5225_01_000001",
			hasError:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseContainerID(tt.containerID)
			if tt.hasError {
				assert.Equal(t, nm.ContainerId{}, result)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
