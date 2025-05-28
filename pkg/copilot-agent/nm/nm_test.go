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
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	statesinformer "github.com/koordinator-sh/koordinator/pkg/koordlet/statesinformer/impl"
	"github.com/koordinator-sh/yarn-copilot/cmd/yarn-copilot-agent/options"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func Test_NodeMangerOperator_NewNodeMangerOperator(t *testing.T) {
	operator := initNodeManagerOperator()
	klog.Infof("operator: %v", &operator)
	time.Sleep(3 * time.Second)
}

func initNodeManagerOperator() NodeMangerOperator {
	original := runtime.GOOS
	var operator NodeMangerOperator
	var err error
	if original != "darwin" && original != "windows" {
		conf := options.NewConfiguration()
		conf.SyncMemoryCgroup = true
		conf.CgroupRootDir = "/tmp"
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensures cleanup after test
		operator, err = NewNodeMangerOperator(conf.CgroupRootDir, conf.YarnContainerCgroupPath, conf.SyncMemoryCgroup, conf.NodeMangerEndpoint, conf.SyncCgroupPeriod, nil)
		if err != nil {
			klog.Fatal(err)
		}
		go func() {
			if err := operator.Run(ctx.Done()); err != nil {
				klog.Error(err)
			}
		}()
	}
	return operator
}

func Test_NodeMangerOperator_KillContainer(t *testing.T) {
	containerId := "container_e517_1746198264116_5225_01_000003"
	operator := initNodeManagerOperator()
	if operator != nil {
		err := operator.KillContainer(containerId)
		assert.Error(t, err)
	}
}

func Test_NodeMangerOperator_ListContainers(t *testing.T) {
	go initHttpServer()
	time.Sleep(2 * time.Second)
	operator := initNodeManagerOperator()
	if operator != nil {
		containers, _ := operator.ListContainers()
		if containers != nil {
			assert.Equal(t, createMockResponse(), *containers)
		}
	}
}

func Test_NodeMangerOperator_GetContainer(t *testing.T) {
	go initHttpServer()
	time.Sleep(2 * time.Second)
	containerId := "container_1697376600001_0001_01_000001"
	operator := initNodeManagerOperator()
	if operator != nil {
		container, _ := operator.GetContainer(containerId)
		assert.NotNil(t, container)
	}
}

func Test_NodeMangerOperator_TestServer(t *testing.T) {
	endpoint := "localhost:8042"
	go initHttpServer()
	time.Sleep(3 * time.Second)
	cli := resty.New()
	cli.SetBaseURL(fmt.Sprintf("http://%s", endpoint))
	var res Containers
	resp, _ := cli.R().SetResult(&res).Get("/ws/v1/node/containers")
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	res2 := testFunc(res)
	assert.Equal(t, createMockResponse(), *res2)
	klog.Infof("res: %v", *res2)
}

func testFunc(res Containers) *Containers {
	return &res
}

func Test_NodeMangerOperator_GenerateCgroupPath(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		expected    string
	}{
		{
			name:        "生成有效的cgroup路径",
			containerID: "container_123",
			expected:    "yarn/container_123",
		},
	}

	operator := &nodeMangerOperator{
		CgroupPath: "yarn",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := operator.GenerateCgroupPath(tt.containerID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_NodeMangerOperator_GenerateCgroupFullPath(t *testing.T) {
	tests := []struct {
		name            string
		cgroupSubSystem string
		expected        string
	}{
		{
			name:            "生成CPU cgroup完整路径",
			cgroupSubSystem: "cpu",
			expected:        "/sys/fs/cgroup/cpu/yarn",
		},
		{
			name:            "生成内存cgroup完整路径",
			cgroupSubSystem: "memory",
			expected:        "/sys/fs/cgroup/memory/yarn",
		},
	}

	operator := &nodeMangerOperator{
		CgroupRoot: "/sys/fs/cgroup",
		CgroupPath: "yarn",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := operator.GenerateCgroupFullPath(tt.cgroupSubSystem)
			assert.Equal(t, tt.expected, result)
		})
	}
}

var (
	httpServerOnce sync.Once
	httpServer     *http.Server
)

func initHttpServer() {
	httpServerOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws/v1/node/containers", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("收到请求: %s %s", r.Method, r.URL.Path)

			response := createMockResponse()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-YARN-Version", "3.3.4")
			w.Header().Set("X-Request-ID", fmt.Sprintf("yarn-req-%d", time.Now().UnixNano()))

			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("JSON编码错误: %v", err)
				http.Error(w, "内部服务器错误", http.StatusInternalServerError)
			}
		})

		httpServer = &http.Server{
			Addr:    ":8042",
			Handler: mux, // 使用自定义路由器
		}

		go func() {
			log.Printf("YARN 节点容器API模拟器启动，监听 http://localhost%s", httpServer.Addr)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("服务器错误: %v", err)
			}
		}()
	})
}

func createMockContainers() []YarnContainer {
	return []YarnContainer{
		{
			Id:                  "container_1697376600001_0001_01_000001",
			Appid:               "application_1697376600001_0001",
			State:               "RUNNING",
			ExitCode:            -1000,
			Diagnostics:         "",
			User:                "user1",
			TotalMemoryNeededMB: 4096,
			TotalVCoresNeeded:   2,
			ContainerLogsLink:   "http://node01:8042/logs/application_1697376600001_0001/container_1697376600001_0001_01_000001",
			NodeId:              "node01.example.com:8041",
			MemUsed:             2456.78,
			MemMaxed:            4096,
			CpuUsed:             1.75,
			CpuMaxed:            2.0,
			ContainerLogFiles:   []string{"syslog", "stderr", "stdout"},
		},
		{
			Id:                  "container_1697376600001_0001_01_000002",
			Appid:               "application_1697376600001_0001",
			State:               "COMPLETED",
			ExitCode:            0,
			Diagnostics:         "Success",
			User:                "user2",
			TotalMemoryNeededMB: 2048,
			TotalVCoresNeeded:   1,
			ContainerLogsLink:   "http://node01:8042/logs/application_1697376600001_0001/container_1697376600001_0001_01_000002",
			NodeId:              "node01.example.com:8041",
			MemUsed:             1984.32,
			MemMaxed:            2048,
			CpuUsed:             0.92,
			CpuMaxed:            1.0,
			ContainerLogFiles:   []string{"syslog", "stderr"},
		},
		{
			Id:                  "container_1697376600001_0002_01_000001",
			Appid:               "application_1697376600001_0002",
			State:               "FAILED",
			ExitCode:            137,
			Diagnostics:         "Container killed on request. Exit code is 137",
			User:                "user3",
			TotalMemoryNeededMB: 8192,
			TotalVCoresNeeded:   4,
			ContainerLogsLink:   "http://node01:8042/logs/application_1697376600001_0002/container_1697376600001_0002_01_000001",
			NodeId:              "node01.example.com:8041",
			MemUsed:             7892.45,
			MemMaxed:            8192,
			CpuUsed:             3.82,
			CpuMaxed:            4.0,
			ContainerLogFiles:   []string{"syslog", "stderr", "stdout", "gc.log"},
		},
		{
			Id:                  "container_1697376600001_0003_01_000001",
			Appid:               "application_1697376600001_0003",
			State:               "NEW",
			ExitCode:            -1000,
			Diagnostics:         "",
			User:                "user4",
			TotalMemoryNeededMB: 1024,
			TotalVCoresNeeded:   1,
			ContainerLogsLink:   "http://node01:8042/logs/application_1697376600001_0003/container_1697376600001_0003_01_000001",
			NodeId:              "node01.example.com:8041",
			MemUsed:             0,
			MemMaxed:            1024,
			CpuUsed:             0,
			CpuMaxed:            1.0,
			ContainerLogFiles:   []string{},
		},
	}
}

// 创建模拟响应
func createMockResponse() Containers {
	containers := createMockContainers()

	var resp Containers
	resp.Containers.Items = containers
	return resp
}

func Test_nodeMangerOperator_ensureCgroupDir(t *testing.T) {
	var dir = "/tmp/cpu/kubepods/besteffort/hadoop-yarn"
	n := &nodeMangerOperator{}
	assert.NoError(t, n.ensureCgroupDir(dir))
}

func Test_nodeMangerOperator_syncNoneProcCgroup(t *testing.T) {
	n := &nodeMangerOperator{}
	n.syncNoneProcCgroup()
}

func Test_nodeMangerOperator_syncAllCgroup(t *testing.T) {
	n := &nodeMangerOperator{}
	n.syncAllCgroup()
}

func Test_nodeMangerOperator_syncParentCgroup(t *testing.T) {
	go initHttpServer()
	time.Sleep(2 * time.Second)
	cgroupRoot := "/tmp"
	endpoint := "localhost:8042"
	filePath := "/tmp/cpu/cpu.shares"
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()
	cli := resty.New()
	cli.SetBaseURL(fmt.Sprintf("http://%s", endpoint))
	n := &nodeMangerOperator{
		CgroupRoot: cgroupRoot,
		client:     cli,
	}
	assert.NoError(t, n.syncParentCgroup())
}

func Test_nodeMangerOperator_syncNMEndpoint(t *testing.T) {
	kubelet, _ := statesinformer.NewKubeletStub("127.0.0.1", 10250, "https", time.Second*5, nil)
	w := NewNMPodWater(kubelet)
	n := &nodeMangerOperator{
		nmPodWatcher: w,
	}
	n.syncNMEndpoint()
}

func Test_nodeMangerOperator_removeMemoryCgroup(t *testing.T) {
	dir := "/tmp/memory/container_1697376600001_0003_01_000001"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}
	cgroupRoot := "/tmp"
	n := &nodeMangerOperator{
		CgroupRoot: cgroupRoot,
	}
	n.removeMemoryCgroup(dir)
}

func Test_nodeMangerOperator_createMemoryCgroup(t *testing.T) {
	dir := "/tmp/memory/container_1697376600001_0003_01_000001"
	os.Remove(dir)
	cgroupRoot := "/tmp"
	n := &nodeMangerOperator{
		CgroupRoot: cgroupRoot,
	}
	n.createMemoryCgroup(dir)
}
