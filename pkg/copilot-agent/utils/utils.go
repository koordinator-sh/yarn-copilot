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

package utils

func DiffMap(m1, m2 map[string]struct{}) (m1More, m1Less map[string]struct{}) {
	m1More = map[string]struct{}{}
	m1Less = map[string]struct{}{}
	for k1 := range m1 {
		if _, ok := m2[k1]; !ok {
			m1More[k1] = struct{}{} // key 仅在第一个 map 中存在
		}
	}

	for k2 := range m2 {
		if _, ok := m1[k2]; !ok {
			m1Less[k2] = struct{}{} // key 仅在第二个 map 中存在
		}
	}

	return
}
