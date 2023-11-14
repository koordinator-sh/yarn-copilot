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

package noderesource

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"reflect"
	"testing"
)

func TestGetOriginExtendAllocatable(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    corev1.ResourceList
		wantErr bool
	}{
		{
			name: "annotation not exist",
			args: args{
				annotations: map[string]string{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "bad annotation format",
			args: args{
				annotations: map[string]string{
					NodeOriginExtendedAllocatableAnnotationKey: "bad-format",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "get from annotation succ",
			args: args{
				annotations: map[string]string{
					NodeOriginExtendedAllocatableAnnotationKey: "{\"resources\": {\"kubernetes.io/batch-cpu\": 1000,\"kubernetes.io/batch-memory\": 1024}}",
				},
			},
			want: map[corev1.ResourceName]resource.Quantity{
				BatchCPU:    resource.MustParse("1000"),
				BatchMemory: resource.MustParse("1024"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetOriginExtendedAllocatableRes(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOriginExtendAllocatableRes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOriginExtendAllocatableRes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNodeAllocated(t *testing.T) {
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    corev1.ResourceList
		wantErr bool
	}{
		{
			name: "annotation not exist",
			args: args{
				annotations: map[string]string{},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "bad annotation format",
			args: args{
				annotations: map[string]string{
					NodeThirdPartyAllocationsAnnotationKey: "bad-format",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "get from annotation succ",
			args: args{
				annotations: map[string]string{
					NodeThirdPartyAllocationsAnnotationKey: "{\"allocations\":[{\"name\":\"hadoop-yarn\",\"priority\":\"koord-batch\",\"resources\":{\"kubernetes.io/batch-cpu\":\"1000\",\"kubernetes.io/batch-memory\":\"1024\"}}]}",
				},
			},
			want: corev1.ResourceList{
				BatchCPU:    resource.MustParse("1000"),
				BatchMemory: resource.MustParse("1024"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetYARNAllocatedResource(tt.args.annotations)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeAllocated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetYARNAllocatedResource() got = %v, want %v", got, tt.want)
			}
		})
	}
}
