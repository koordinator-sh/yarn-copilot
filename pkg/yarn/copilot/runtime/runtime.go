package runtime

import (
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func GetAliveCopilots() map[string]CustomRuntimePlugin {
	if defaultManger == nil {
		return map[string]CustomRuntimePlugin{}
	}
	return defaultManger.GetAliveCopilots()
}

func GetCopilot(name string) (CustomRuntimePlugin, bool) {
	if defaultManger == nil {
		return nil, false
	}
	return defaultManger.GetCopilot(name)
}

type ContainerInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	UID             string            `json:"uid"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	CreateTimestamp time.Time         `json:"createTimestamp"`

	CgroupDir   string                  `json:"cgroupDir"`
	HostNetwork bool                    `json:"hostNetwork"`
	Resources   v1.ResourceRequirements `json:"resources"`
}

type PluginInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type KillRequest struct {
	ContainerID string          `json:"containerID,omitempty"`
	Resources   v1.ResourceList `json:"resources,omitempty"`
}

type KillInfo struct {
	Items []*ContainerInfo `json:"items,omitempty"`
}

type CustomRuntimePlugin interface {
	IsAlive() bool
	Info() (*PluginInfo, error)
	ListContainer() ([]*ContainerInfo, error)
	GetContainer(ContainerID string) (*ContainerInfo, error)
	KillContainer(killReq *KillRequest) (*KillInfo, error)
	KillContainersByResource(killReq *KillRequest) (*KillInfo, error)
}

type httpPlugin struct {
	name string

	client *resty.Client
}

func newHttpPlugin(socket string) *httpPlugin {

	transport := http.Transport{
		Dial: func(_, _ string) (net.Conn, error) {
			return net.Dial("unix", socket)
		},
	}

	// Create a Resty Client
	client := resty.New()

	// Set the previous transport that we created, set the scheme of the communication to the
	// socket and set the unixSocket as the HostURL.
	client.SetTransport(&transport).SetScheme("http").SetHostURL("unixSocket")
	return &httpPlugin{
		client: client,
	}
}

func (h *httpPlugin) IsAlive() bool {
	resp, err := h.client.R().Get("/health")
	if err != nil {
		klog.V(4).Info(err.Error())
		return false
	}
	if resp.IsError() {
		klog.V(4).Info(resp.Error())
		return false
	}
	klog.V(5).Infof("health from plugin, %+v", string(resp.Body()))
	return resp.IsSuccess()
}

func (h *httpPlugin) Info() (*PluginInfo, error) {
	var res *PluginInfo
	_, err := h.client.R().
		SetResult(&res).
		Get("/information")
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *httpPlugin) ListContainer() ([]*ContainerInfo, error) {
	var res []*ContainerInfo
	_, err := h.client.R().SetResult(&res).Get("/v1/containers")
	klog.V(5).Infof("list container from plugin, %+v", res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *httpPlugin) GetContainer(ContainerID string) (*ContainerInfo, error) {
	var res *ContainerInfo
	_, err := h.client.R().
		SetResult(&res).
		SetQueryParam("containerID", ContainerID).
		Get("/v1/container")
	klog.V(5).Infof("get container from plugin, %+v", res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *httpPlugin) KillContainer(killReq *KillRequest) (*KillInfo, error) {
	var res *KillInfo
	_, err := h.client.R().
		SetResult(&res).
		SetBody(killReq).
		Post("/v1/killContainer")
	klog.V(5).Infof("kill container from plugin, %+v", res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *httpPlugin) KillContainersByResource(killReq *KillRequest) (*KillInfo, error) {
	var res *KillInfo
	_, err := h.client.R().
		SetResult(&res).
		SetBody(killReq).
		Post("/v1/killContainersByResource")
	if err != nil {
		return nil, err
	}
	return res, nil
}
