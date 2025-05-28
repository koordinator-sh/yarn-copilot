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
	"context"
	"errors"
	"fmt"
	"github.com/koordinator-sh/koordinator/apis/extension"
	"k8s.io/apimachinery/pkg/api/resource"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/koordinator-sh/koordinator/pkg/koordlet/util/system"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/koordinator-sh/yarn-copilot/pkg/copilot-agent/nm"
)

type YarnCopilotServer struct {
	mgr      nm.NodeMangerOperator
	unixPath string
}

func NewYarnCopilotServer(mgr nm.NodeMangerOperator, unixPath string) *YarnCopilotServer {
	return &YarnCopilotServer{mgr: mgr, unixPath: unixPath}
}

func (y *YarnCopilotServer) Run(ctx context.Context) error {
	e := gin.New()
	e.GET("/health", y.Health)
	e.GET("/information", y.Information)
	e.GET("/v1/container", y.GetContainer)
	e.GET("/v1/containers", y.ListContainers)
	e.POST("/v1/killContainer", y.KillContainer)
	e.POST("/v1/killContainersByResource", y.KillContainersByResource)

	server := &http.Server{
		Handler: e,
	}
	sockDir := filepath.Dir(y.unixPath)
	_ = os.MkdirAll(sockDir, os.ModePerm)
	if system.FileExists(y.unixPath) {
		_ = os.Remove(y.unixPath)
	}
	listener, err := net.Listen("unix", y.unixPath)
	if err != nil {
		fmt.Printf("Failed to listen UNIX socket: %v", err)
		os.Exit(1)
	}
	defer func() {
		_ = os.Remove(y.unixPath)
	}()
	go func() {
		_ = server.Serve(listener)
	}()
	//for {
	//	select {
	//	case <-ctx.Done():
	//
	//	}
	//}
	for range ctx.Done() {
		klog.Info("graceful shutdown")
		if err := server.Shutdown(ctx); err != nil {
			klog.Errorf("Server forced to shutdown: %v", err)
			return err
		}
	}
	return nil
}

func (y *YarnCopilotServer) Health(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, "ok")
}

type PluginInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (y *YarnCopilotServer) Information(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, &PluginInfo{
		Name:    "yarn",
		Version: "v1",
	})
}

func (y *YarnCopilotServer) ListContainers(ctx *gin.Context) {
	listContainers, err := y.mgr.ListContainers()
	if err != nil {
		klog.Error(err)
		ctx.JSON(http.StatusBadRequest, err)
		return
	}
	res := make([]*ContainerInfo, 0, len(listContainers.Containers.Items))
	for _, container := range listContainers.Containers.Items {
		if container.IsFinalState() {
			continue
		}
		res = append(res, ParseContainerInfo(&container, y.mgr))
	}
	ctx.JSON(http.StatusOK, res)
}

func (y *YarnCopilotServer) GetContainer(ctx *gin.Context) {
	containerID := ctx.Query("containerID")
	container, err := y.mgr.GetContainer(containerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, err)
		return
	}
	ctx.JSON(http.StatusOK, ParseContainerInfo(container, y.mgr))
}

type KillRequest struct {
	ContainerID string          `json:"containerID,omitempty"`
	Resources   v1.ResourceList `json:"resources,omitempty"`
}

type KillInfo struct {
	Items []*ContainerInfo `json:"items,omitempty"`
}

type ContainerInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	UID             string            `json:"uid"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	Priority        int32             `json:"priority"`
	CreateTimestamp time.Time         `json:"createTimestamp"`

	CgroupDir   string                  `json:"cgroupDir"`
	HostNetwork bool                    `json:"hostNetwork"`
	Resources   v1.ResourceRequirements `json:"resources"`
}

func (y *YarnCopilotServer) KillContainer(ctx *gin.Context) {
	var kr KillRequest
	if err := ctx.BindJSON(&kr); err != nil {
		ctx.JSON(http.StatusBadRequest, err)
		return
	}
	container, err := y.mgr.GetContainer(kr.ContainerID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	if err := y.mgr.KillContainer(kr.ContainerID); err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, KillInfo{Items: []*ContainerInfo{ParseContainerInfo(container, y.mgr)}})
}

func (y *YarnCopilotServer) KillContainersByResource(ctx *gin.Context) {
	var kr KillRequest
	if err := ctx.BindJSON(&kr); err != nil {
		ctx.JSON(http.StatusBadRequest, err)
		return
	}
	klog.Info("KillRequest: %s", kr)
	res, err := y.mgr.ListContainers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	needReleasedCpu, _ := kr.Resources.Name(extension.BatchCPU, resource.DecimalSI).AsInt64()
	needReleasedMemory, _ := kr.Resources.Name(extension.BatchMemory, resource.BinarySI).AsInt64()
	var currentReleasedCpu, currentReleasedMemory int
	containers := res.Containers.Items
	if len(containers) > 0 {
		filteredContainers := filter(containers)
		if len(filteredContainers) == 0 {
			filteredContainers = containers
		}
		scoredContainers := score(filteredContainers)
		for _, container := range scoredContainers {
			if err := y.mgr.KillContainer(container.Id); err != nil {
				klog.Errorf("KillContainersByResource error: %s", container)
				ctx.JSON(http.StatusInternalServerError, err)
				return
			} else {
				klog.Infof("kill container %s", container)
				currentReleasedCpu += container.TotalVCoresNeeded * 1000
				currentReleasedMemory += container.TotalMemoryNeededMB * 1024 * 1024
				if int64(currentReleasedCpu) >= needReleasedCpu && int64(currentReleasedMemory) >= needReleasedMemory {
					break
				}
			}
		}
	}
	releasedResourceList := v1.ResourceList{
		extension.BatchCPU:    *resource.NewMilliQuantity(int64(currentReleasedCpu), resource.DecimalSI),
		extension.BatchMemory: *resource.NewQuantity(int64(currentReleasedMemory), resource.BinarySI),
	}
	klog.Infof("release resources: %s", releasedResourceList)
	ctx.JSON(http.StatusOK, releasedResourceList)
}

func filter(containers []nm.YarnContainer) []nm.YarnContainer {
	var filtered []nm.YarnContainer
	for _, c := range containers {
		lastSeg, err := parseLastSegment(c.Id)
		if err != nil || lastSeg != "000001" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func score(containers []nm.YarnContainer) []nm.YarnContainer {
	if len(containers) > 0 {
		sort.Sort(ReverseContainerSorter(containers))
		klog.V(4).Infof("Sorted containers by ID in descending order: %+v", containers)
	}
	return containers
}

/**
 * containerId
 * container_e*epoch*_*clusterTimestamp*_*appId*_*attemptId*_*containerId*
 * container_*clusterTimestamp*_*appId*_*attemptId*_*containerId*
 */
func parseLastSegment(containerID string) (string, error) {
	if containerID == "" {
		return "", errors.New("container ID 不能为空")
	}
	segments := strings.Split(containerID, "_")
	if (len(segments) != 5 && len(segments) != 6) || segments[0] != "container" {
		return "", fmt.Errorf("无效的容器ID格式: %s", containerID)
	}
	lastSegment := segments[len(segments)-1]
	return lastSegment, nil
}

type ReverseContainerSorter []nm.YarnContainer

func (cs ReverseContainerSorter) Len() int      { return len(cs) }
func (cs ReverseContainerSorter) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs ReverseContainerSorter) Less(i, j int) bool {
	a, b := cs[i], cs[j]
	containerA, _ := parseContainerID(a.Id)
	containerB, _ := parseContainerID(b.Id)
	switch {
	case containerA.ClusterTS != containerB.ClusterTS:
		return containerA.ClusterTS > containerB.ClusterTS
	case containerA.AppID != containerB.AppID:
		return containerA.AppID > containerB.AppID
	case containerA.AttemptID != containerB.AttemptID:
		return containerA.AttemptID > containerB.AttemptID
	default:
		return containerA.ContainerID > containerB.ContainerID
	}
}

func parseContainerID(id string) (nm.ContainerId, error) {
	segments := strings.Split(id, "_")
	if (len(segments) != 5 && len(segments) != 6) || segments[0] != "container" {
		return nm.ContainerId{}, fmt.Errorf("invalid container ID format: %s", id)
	}

	clusterTSStr := segments[len(segments)-4]
	clusterTS, err := strconv.ParseInt(clusterTSStr, 10, 64)
	if 13 != len(clusterTSStr) || err != nil {
		return nm.ContainerId{}, fmt.Errorf("invalid cluster timestamp: %s", clusterTSStr)

	}

	appID, err := strconv.ParseInt(segments[len(segments)-3], 10, 64)
	if err != nil {
		return nm.ContainerId{}, fmt.Errorf("invalid app ID: %s", segments[len(segments)-3])
	}

	attemptID, err := strconv.Atoi(segments[len(segments)-2])
	if err != nil {
		return nm.ContainerId{}, fmt.Errorf("invalid attempt ID: %s", segments[len(segments)-2])
	}

	containerID, err := strconv.Atoi(segments[len(segments)-1])
	if err != nil {
		return nm.ContainerId{}, fmt.Errorf("invalid container ID: %s", segments[len(segments)-1])
	}

	return nm.ContainerId{
		ID:          id,
		ClusterTS:   clusterTS,
		AppID:       appID,
		AttemptID:   attemptID,
		ContainerID: containerID,
	}, nil
}
