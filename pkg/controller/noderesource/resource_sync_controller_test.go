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
	"context"
	"reflect"
	"testing"

	"github.com/golangplus/testing/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/koordinator-sh/goyarn/pkg/yarn/cache"
)

func TestYARNResourceSyncReconciler_getYARNNode(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	type fields struct {
		pods *corev1.Pod
	}
	type args struct {
		node *corev1.Node
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *cache.YarnNode
		wantErr bool
	}{
		{
			name: "default yarn node",
			fields: fields{
				pods: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node-manager",
						Labels: map[string]string{
							YarnNMComponentLabel: YarnNMComponentValue,
						},
						Annotations: map[string]string{
							YarnNodeIdAnnotation:    "test-nm-id:8041",
							YarnClusterIDAnnotation: "test-cluster-id",
						},
					},
					Spec: corev1.PodSpec{
						NodeName: "test-node-name",
					},
				},
			},
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node-name",
					},
				},
			},
			want: &cache.YarnNode{
				Name:      "test-nm-id",
				Port:      8041,
				ClusterID: "test-cluster-id",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &YARNResourceSyncReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			}
			if tt.fields.pods != nil {
				err := r.Client.Create(context.Background(), tt.fields.pods)
				assert.NoError(t, err)
			}

			got, err := r.getYARNNode(tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("getYARNNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getYARNNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}
