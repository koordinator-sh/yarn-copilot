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
	tick := time.NewTicker(time.Minute)
	for {
		select {
		case <-tick.C:
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
		pluginInfo, err1 := p.Info()
		if err1 != nil {
			return err1
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
