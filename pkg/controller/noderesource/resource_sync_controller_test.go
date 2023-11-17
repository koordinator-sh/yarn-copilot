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
	"fmt"
	"github.com/golang/mock/gomock"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/cache"
	yarnclient "github.com/koordinator-sh/yarn-copilot/pkg/yarn/client"
	"github.com/koordinator-sh/yarn-copilot/pkg/yarn/client/mockclient"
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
							YarnNodeIdAnnotation:          "test-nm-id:8041",
							PodYarnClusterIDAnnotationKey: "test-cluster-id",
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

func Test_getNodeBatchResource(t *testing.T) {
	type args struct {
		node              *corev1.Node
		originAllocatable corev1.ResourceList
	}
	tests := []struct {
		name            string
		args            args
		wantBatchCPU    resource.Quantity
		wantBatchMemory resource.Quantity
		wantErr         bool
	}{
		{
			name: "get from node status",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{},
					Status: corev1.NodeStatus{
						Allocatable: map[corev1.ResourceName]resource.Quantity{
							BatchCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
							BatchMemory: *resource.NewQuantity(1024, resource.BinarySI),
						},
					},
				},
			},
			wantBatchCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
			wantBatchMemory: *resource.NewQuantity(1024, resource.BinarySI),
			wantErr:         false,
		},
		{
			name: "get from node annotation",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
					Status: corev1.NodeStatus{},
				},
				originAllocatable: corev1.ResourceList{
					BatchCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
					BatchMemory: *resource.NewQuantity(1024, resource.BinarySI),
				},
			},
			wantBatchCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
			wantBatchMemory: *resource.NewQuantity(1024, resource.BinarySI),
			wantErr:         false,
		},
		{
			name: "get from node annotation even node status exist",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
					Status: corev1.NodeStatus{
						Allocatable: map[corev1.ResourceName]resource.Quantity{
							BatchCPU:    *resource.NewQuantity(2000, resource.DecimalSI),
							BatchMemory: *resource.NewQuantity(20484, resource.BinarySI),
						},
					},
				},
				originAllocatable: corev1.ResourceList{
					BatchCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
					BatchMemory: *resource.NewQuantity(1024, resource.BinarySI),
				},
			},
			wantBatchCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
			wantBatchMemory: *resource.NewQuantity(1024, resource.BinarySI),
			wantErr:         false,
		},
		{
			name: "return zero with nil node",
			args: args{
				node: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.originAllocatable != nil && tt.args.node != nil && tt.args.node.Annotations != nil {
				err := SetOriginExtendedAllocatableRes(tt.args.node.Annotations, tt.args.originAllocatable)
				assert.NoError(t, err)
			}
			gotBatchCPU, gotBatchMemory, err := getNodeBatchResource(tt.args.node)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantBatchCPU.MilliValue(), gotBatchCPU.MilliValue())
			assert.Equal(t, tt.wantBatchMemory.MilliValue(), gotBatchMemory.MilliValue())
		})
	}
}

func TestYARNResourceSyncReconciler_updateYARNAllocatedResource(t *testing.T) {
	type args struct {
		node     *corev1.Node
		vcores   int32
		memoryMB int64
	}
	tests := []struct {
		name          string
		args          args
		wantAllocated corev1.ResourceList
		wantErr       bool
	}{
		{
			name:          "nil node do nothing",
			args:          args{},
			wantAllocated: nil,
			wantErr:       false,
		},
		{
			name: "update node allocated",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-node",
						Annotations: map[string]string{},
					},
				},
				vcores:   1,
				memoryMB: 1024,
			},
			wantAllocated: corev1.ResourceList{
				BatchCPU:    resource.MustParse("1k"),
				BatchMemory: resource.MustParse("1Gi"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			assert.NoError(t, clientgoscheme.AddToScheme(scheme))
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &YARNResourceSyncReconciler{
				Client: client,
			}
			if tt.args.node != nil {
				assert.NoError(t, r.Client.Create(context.TODO(), tt.args.node))
			}
			err := r.updateYARNAllocatedResource(tt.args.node, tt.args.vcores, tt.args.memoryMB)
			assert.Equal(t, err != nil, tt.wantErr)
			if tt.args.node != nil {
				gotNode := &corev1.Node{}
				key := types.NamespacedName{Name: tt.args.node.Name}
				assert.NoError(t, r.Client.Get(context.TODO(), key, gotNode))
				nodeAllocated, gotErr := GetYARNAllocatedResource(gotNode.Annotations)
				assert.Equal(t, tt.wantAllocated, nodeAllocated)
				assert.NoError(t, gotErr)
			}
		})
	}
}

func TestYARNResourceSyncReconciler_getYARNNodeManagerPod(t *testing.T) {
	type args struct {
		node *corev1.Node
		pods []*corev1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Pod
		wantErr bool
	}{
		{
			name: "nil node with empty return",
			args: args{
				node: nil,
				pods: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "get node manager pod",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-nm-pod",
							Labels: map[string]string{
								YarnNMComponentLabel: YarnNMComponentValue,
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "test-node",
						},
					},
				},
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-nm-pod",
					Labels: map[string]string{
						YarnNMComponentLabel: YarnNMComponentValue,
					},
					ResourceVersion: "1",
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
				},
			},
			wantErr: false,
		},
		{
			name: "node manager node found because of node label",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:   "test-nm-pod",
							Labels: map[string]string{},
						},
						Spec: corev1.PodSpec{
							NodeName: "test-node",
						},
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			assert.NoError(t, clientgoscheme.AddToScheme(scheme))
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &YARNResourceSyncReconciler{
				Client: client,
			}
			if tt.args.node != nil {
				assert.NoError(t, r.Client.Create(context.TODO(), tt.args.node))
			}
			for _, pod := range tt.args.pods {
				assert.NoError(t, r.Client.Create(context.TODO(), pod))
			}
			got, err := r.getYARNNodeManagerPod(tt.args.node)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestYARNResourceSyncReconciler_getYARNNode1(t *testing.T) {
	type args struct {
		node *corev1.Node
		pods []*corev1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    *cache.YarnNode
		wantErr bool
	}{
		{
			name: "nil node return empty",
			args: args{
				node: nil,
				pods: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "get yarn node",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-nm-pod",
							Labels: map[string]string{
								YarnNMComponentLabel: YarnNMComponentValue,
							},
							Annotations: map[string]string{
								YarnNodeIdAnnotation:          "test-yarn-node-id:8042",
								PodYarnClusterIDAnnotationKey: "test-yarn-cluster-id",
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "test-node",
						},
					},
				},
			},
			want: &cache.YarnNode{
				Name:      "test-yarn-node-id",
				Port:      8042,
				ClusterID: "test-yarn-cluster-id",
			},
			wantErr: false,
		},
		{
			name: "yarn node id not exist",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-nm-pod",
							Labels: map[string]string{
								YarnNMComponentLabel: YarnNMComponentValue,
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "test-node",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "yarn node id bad format",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-nm-pod",
							Labels: map[string]string{
								YarnNMComponentLabel: YarnNMComponentValue,
							},
							Annotations: map[string]string{
								YarnNodeIdAnnotation:          "test-yarn-node-id-8042",
								PodYarnClusterIDAnnotationKey: "test-yarn-cluster-id",
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "test-node",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "yarn node id port bad format",
			args: args{
				node: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
				},
				pods: []*corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-nm-pod",
							Labels: map[string]string{
								YarnNMComponentLabel: YarnNMComponentValue,
							},
							Annotations: map[string]string{
								YarnNodeIdAnnotation:          "test-yarn-node-id:bad-port",
								PodYarnClusterIDAnnotationKey: "test-yarn-cluster-id",
							},
						},
						Spec: corev1.PodSpec{
							NodeName: "test-node",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			assert.NoError(t, clientgoscheme.AddToScheme(scheme))
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &YARNResourceSyncReconciler{
				Client: client,
			}
			if tt.args.node != nil {
				assert.NoError(t, r.Client.Create(context.TODO(), tt.args.node))
			}
			for _, pod := range tt.args.pods {
				assert.NoError(t, r.Client.Create(context.TODO(), pod))
			}
			got, err := r.getYARNNode(tt.args.node)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestYARNResourceSyncReconciler_getYARNClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	yarnClient := mock_client.NewMockYarnClient(ctrl)
	yarnCluster1Client := mock_client.NewMockYarnClient(ctrl)
	yarnClients := map[string]yarnclient.YarnClient{
		"test-cluster1": yarnCluster1Client,
	}

	type fields struct {
		yarnClientOfReconciler  yarnclient.YarnClient
		yarnClientsOfReconciler map[string]yarnclient.YarnClient
		yarnClientFromFactory   yarnclient.YarnClient
		yarnClientsFromFactory  map[string]yarnclient.YarnClient
	}
	type args struct {
		yarnNode                  *cache.YarnNode
		factoryCreateDefaultError error
		factoryCreateClusterError map[string]error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    yarnclient.YarnClient
		wantErr bool
	}{
		{
			name:   "return nil with nil node",
			fields: fields{},
			args: args{
				yarnNode: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "get default client by empty cluster id",
			fields: fields{
				yarnClientOfReconciler: yarnClient,
			},
			args: args{
				yarnNode: &cache.YarnNode{
					Name:      "test-node",
					Port:      8042,
					ClusterID: "",
				},
			},
			want:    yarnClient,
			wantErr: false,
		},
		{
			name: "get default client by empty cluster id from new",
			fields: fields{
				yarnClientFromFactory: yarnClient,
			},
			args: args{
				yarnNode: &cache.YarnNode{
					Name:      "test-node",
					Port:      8042,
					ClusterID: "",
				},
			},
			want:    yarnClient,
			wantErr: false,
		},
		{
			name: "get default client by empty cluster id from new with error",
			fields: fields{
				yarnClientFromFactory: yarnClient,
			},
			args: args{
				yarnNode: &cache.YarnNode{
					Name:      "test-node",
					Port:      8042,
					ClusterID: "",
				},
				factoryCreateDefaultError: fmt.Errorf("create default error"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "get cluster client from reconciler",
			fields: fields{
				yarnClientsOfReconciler: yarnClients,
			},
			args: args{
				yarnNode: &cache.YarnNode{
					Name:      "test-node",
					Port:      8042,
					ClusterID: "test-cluster1",
				},
			},
			want:    yarnCluster1Client,
			wantErr: false,
		},
		{
			name: "get cluster client from new",
			fields: fields{
				yarnClientsFromFactory: yarnClients,
			},
			args: args{
				yarnNode: &cache.YarnNode{
					Name:      "test-node",
					Port:      8042,
					ClusterID: "test-cluster1",
				},
			},
			want:    yarnCluster1Client,
			wantErr: false,
		},
		{
			name: "get cluster client from new with error",
			fields: fields{
				yarnClientsFromFactory: yarnClients,
			},
			args: args{
				yarnNode: &cache.YarnNode{
					Name:      "test-node",
					Port:      8042,
					ClusterID: "test-cluster1",
				},
				factoryCreateClusterError: map[string]error{
					"test-cluster1": fmt.Errorf("create default error"),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockYarnClientFactory := mock_client.NewMockYarnClientFactory(ctrl)
			yarnclient.DefaultYarnClientFactory = mockYarnClientFactory

			if tt.fields.yarnClientFromFactory != nil {
				mockYarnClientFactory.EXPECT().CreateDefaultYarnClient().Return(tt.fields.yarnClientFromFactory, tt.args.factoryCreateDefaultError)
			}
			for clusterID, client := range tt.fields.yarnClientsFromFactory {
				createError := tt.args.factoryCreateClusterError[clusterID]
				mockYarnClientFactory.EXPECT().CreateYarnClientByClusterID(clusterID).Return(client, createError)
			}
			r := &YARNResourceSyncReconciler{
				yarnClient:  tt.fields.yarnClientOfReconciler,
				yarnClients: tt.fields.yarnClientsOfReconciler,
			}
			got, err := r.getYARNClient(tt.args.yarnNode)
			if (err != nil) != tt.wantErr {
				t.Errorf("getYARNClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getYARNClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestYARNResourceSyncReconciler_updateYARNNodeResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type fields struct {
		yarnClientErrorFromFactory error
		doUpdate                   bool
		updateNodeResourceError    error
		doReinit                   bool
		reinitError                error
	}
	type args struct {
		yarnNode *cache.YarnNode
		vcores   int64
		memoryMB int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "nil node return nil",
			fields: fields{
				doUpdate: false,
			},
			args:    args{},
			wantErr: false,
		},
		{
			name: "get client failed",
			fields: fields{
				yarnClientErrorFromFactory: fmt.Errorf("create client error"),
			},
			args: args{
				yarnNode: &cache.YarnNode{},
			},
			wantErr: true,
		},
		{
			name: "update succ",
			fields: fields{
				doUpdate: true,
			},
			args: args{
				yarnNode: &cache.YarnNode{},
			},
			wantErr: false,
		},
		{
			name: "update failed",
			fields: fields{
				doUpdate:                true,
				updateNodeResourceError: fmt.Errorf("update error"),
				doReinit:                true,
			},
			args: args{
				yarnNode: &cache.YarnNode{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockYarnClientFactory := mock_client.NewMockYarnClientFactory(ctrl)
			yarnclient.DefaultYarnClientFactory = mockYarnClientFactory
			yarnClient := mock_client.NewMockYarnClient(ctrl)
			if tt.args.yarnNode != nil {
				mockYarnClientFactory.EXPECT().CreateDefaultYarnClient().Return(yarnClient, tt.fields.yarnClientErrorFromFactory)
			}
			if tt.fields.doUpdate {
				yarnClient.EXPECT().UpdateNodeResource(gomock.Any()).Return(nil, tt.fields.updateNodeResourceError)
			}
			if tt.fields.doReinit {
				yarnClient.EXPECT().Reinitialize().Return(tt.fields.reinitError)
			}

			r := &YARNResourceSyncReconciler{}
			if err := r.updateYARNNodeResource(tt.args.yarnNode, tt.args.vcores, tt.args.memoryMB); (err != nil) != tt.wantErr {
				t.Errorf("updateYARNNodeResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestYARNResourceSyncReconciler_getYARNNodeAllocatedResource(t *testing.T) {
	type args struct {
		yarnNode *cache.YarnNode
	}
	tests := []struct {
		name         string
		args         args
		wantVcores   int32
		wantMemoryMB int64
		wantErr      bool
	}{
		{
			name: "nil node return nothing",
			args: args{
				yarnNode: &cache.YarnNode{
					Name: "test-yarn-node",
					Port: 8041,
				},
			},
			wantVcores:   0,
			wantMemoryMB: 0,
			wantErr:      false,
		},
		{
			name: "get yarn node not exist",
			args: args{
				yarnNode: &cache.YarnNode{
					Name: "test-yarn-node",
					Port: 8041,
				},
			},
			wantVcores:   0,
			wantMemoryMB: 0,
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockYarnClientFactory := mock_client.NewMockYarnClientFactory(ctrl)
			yarnclient.DefaultYarnClientFactory = mockYarnClientFactory
			yarnClient := mock_client.NewMockYarnClient(ctrl)
			yarnNodeCache := cache.NewNodesSyncer(map[string]yarnclient.YarnClient{"default": yarnClient})

			r := &YARNResourceSyncReconciler{
				yarnNodeCache: yarnNodeCache,
			}
			gotVcores, gotMemoryMB, err := r.getYARNNodeAllocatedResource(tt.args.yarnNode)
			if (err != nil) != tt.wantErr {
				t.Errorf("getYARNNodeAllocatedResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotVcores != tt.wantVcores {
				t.Errorf("getYARNNodeAllocatedResource() gotVcores = %v, want %v", gotVcores, tt.wantVcores)
			}
			if gotMemoryMB != tt.wantMemoryMB {
				t.Errorf("getYARNNodeAllocatedResource() gotMemoryMB = %v, want %v", gotMemoryMB, tt.wantMemoryMB)
			}
		})
	}
}
