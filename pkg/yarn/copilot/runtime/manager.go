package runtime

import (
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

var defaultManger *Manger

type Manger struct {
	SocketDir string

	Plugins map[string]CustomRuntimePlugin

	mtx sync.RWMutex
}

func NewManger(socketDir string) *Manger {
	m := &Manger{SocketDir: socketDir, Plugins: map[string]CustomRuntimePlugin{}, mtx: sync.RWMutex{}}
	defaultManger = m
	return m
}

func (m *Manger) Run(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	tick := time.Tick(time.Minute)
	for {
		select {
		case <-tick:
			if err := m.run(); err != nil {
				klog.Warning(err)
			}
		case <-stopCh:
			break
		}
	}
}

func (m *Manger) run() error {

	klog.V(4).Info("watch socket path %s", m.SocketDir)
	err := filepath.Walk(m.SocketDir, func(path string, info fs.FileInfo, err error) error {

		if info.Mode().Type() != fs.ModeSocket {
			klog.V(4).Infof("%s is not socket", path)
			return nil
		}
		p := newHttpPlugin(path)
		pluginInfo, err := p.Info()
		if err != nil {
			return err
		}
		klog.Infof("discover plugin %s", pluginInfo.Name)
		m.mtx.Lock()
		m.Plugins[pluginInfo.Name] = p
		m.mtx.Unlock()
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *Manger) GetAliveCopilots() map[string]CustomRuntimePlugin {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	res := map[string]CustomRuntimePlugin{}
	for key, plugin := range m.Plugins {
		if plugin.IsAlive() {
			res[key] = plugin
		}
	}
	return res
}

func (m *Manger) GetCopilot(name string) (CustomRuntimePlugin, bool) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	res, exist := m.Plugins[name]
	return res, exist
}
