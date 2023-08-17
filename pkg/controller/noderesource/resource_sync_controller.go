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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/cri-api/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"

	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	yarnserver "github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn/server"
	yarnclient "github.com/koordinator-sh/goyarn/pkg/yarn/client"
	"github.com/koordinator-sh/koordinator/apis/extension"
)

const (
	Name = "yarnresource"
)

type YARNResourceSyncReconciler struct {
	client.Client
	yarnClient  *yarnclient.YarnClient
	yarnClients map[string]*yarnclient.YarnClient
}

func (r *YARNResourceSyncReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	node := &corev1.Node{}

	if err := r.Client.Get(context.TODO(), req.NamespacedName, node); err != nil {
		if errors.IsNotFound(err) {
			klog.V(3).Infof("skip for node %v not found", req.Name)
			return ctrl.Result{}, nil
		}
		klog.Warningf("failed to get node %v, error %v", req.Name, err)
		return ctrl.Result{Requeue: true}, err
	}

	yarnNode, err := r.getYARNNode(node)
	if err != nil {
		klog.Warningf("fail to parse yarn node name for %v, error %v", node.Name, err)
		return ctrl.Result{}, nil
	}
	if yarnNode == nil || yarnNode.Name == "" || yarnNode.Port == 0 {
		klog.V(3).Infof("skip for yarn node not exist on node %v, detail %+v", req.Name, yarnNode)
		return ctrl.Result{}, nil
	}

	// TODO exclude batch pod requested
	batchCPU, cpuExist := node.Status.Allocatable[extension.BatchCPU]
	batchMemory, memExist := node.Status.Allocatable[extension.BatchMemory]
	if !cpuExist || !memExist {
		klog.V(3).Infof("skip sync node %v, since batch cpu or memory not exist in allocatable %v", node.Name, node.Status.Allocatable)
		return ctrl.Result{}, nil
	}

	// TODO control update frequency
	if err := r.updateYARNNodeResource(yarnNode, batchCPU, batchMemory); err != nil {
		klog.Warningf("update batch resource to yarn node %v failed, error %v", yarnNode, err)
		return ctrl.Result{Requeue: true}, err
	}
	klog.V(4).Infof("update node %v batch resource to yarn %+v finish, cpu-core %v, memory-mb %v",
		node.Name, yarnNode, batchCPU.ScaledValue(resource.Kilo), batchMemory.ScaledValue(resource.Mega))
	return ctrl.Result{}, nil
}

func Add(mgr ctrl.Manager) error {
	r := &YARNResourceSyncReconciler{
		Client:      mgr.GetClient(),
		yarnClients: make(map[string]*yarnclient.YarnClient),
	}
	return r.SetupWithManager(mgr)
}

func (r *YARNResourceSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named(Name).
		Complete(r)
}

func (r *YARNResourceSyncReconciler) getYARNNodeManagerPod(node *corev1.Node) (*corev1.Pod, error) {
	opts := []client.ListOption{
		client.MatchingLabels{yarnNMComponentLabel: yarnNMComponentValue},
		client.MatchingFields{"spec.nodeName": node.Name},
	}
	podList := &corev1.PodList{}
	if err := r.Client.List(context.TODO(), podList, opts...); err != nil {
		return nil, fmt.Errorf("get node manager pod failed on node %v with error %v", node.Name, err)
	}
	if len(podList.Items) == 0 {
		return nil, nil
	}
	if len(podList.Items) > 1 {
		klog.Warningf("get %v node manager pod on node %v, will select the first one", len(podList.Items), node.Name)
		return &podList.Items[0], nil
	}
	return &podList.Items[0], nil
}

func (r *YARNResourceSyncReconciler) getYARNNode(node *corev1.Node) (*yarnNode, error) {
	if node == nil {
		return nil, nil
	}

	nmPod, err := r.getYARNNodeManagerPod(node)
	if err != nil {
		return nil, err
	} else if nmPod == nil {
		return nil, nil
	}

	podAnnoNodeId, exists := nmPod.Annotations[yarnNodeIdAnnotation]
	if !exists {
		return nil, fmt.Errorf("yarn nm id %v not exist in node annotationv", yarnNodeIdAnnotation)
	}
	tokens := strings.Split(podAnnoNodeId, ":")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("yarn nm id %v format is illegal", podAnnoNodeId)
	}
	port, err := strconv.Atoi(tokens[1])
	if err != nil {
		return nil, fmt.Errorf("yarn nm id port %v format is illegal", podAnnoNodeId)
	}

	yarnNode := &yarnNode{
		Name: tokens[0],
		Port: int32(port),
	}
	if clusterID, exist := nmPod.Annotations[yarnClusterIDAnnotation]; exist {
		yarnNode.ClusterID = clusterID
	}
	return yarnNode, nil
}

func (r *YARNResourceSyncReconciler) updateYARNNodeResource(yarnNode *yarnNode, cpuMilli, memory resource.Quantity) error {
	// convert to yarn format
	vcores := int32(cpuMilli.ScaledValue(resource.Kilo))
	memoryMB := memory.ScaledValue(resource.Mega)

	request := &yarnserver.UpdateNodeResourceRequestProto{
		NodeResourceMap: []*hadoopyarn.NodeResourceMapProto{
			{
				NodeId: &hadoopyarn.NodeIdProto{
					Host: pointer.String(yarnNode.Name),
					Port: pointer.Int32(yarnNode.Port),
				},
				ResourceOption: &hadoopyarn.ResourceOptionProto{
					Resource: &hadoopyarn.ResourceProto{
						Memory:       &memoryMB,
						VirtualCores: &vcores,
					},
				},
			},
		},
	}
	yarnClient, err := r.getYARNClient(yarnNode)
	if err != nil {
		return err
	}
	if resp, err := yarnClient.UpdateNodeResource(request); err != nil {
		initErr := yarnClient.Reinitialize()
		return fmt.Errorf("UpdateNodeResource resp %v, error %v, reinitialize error %v", resp, err, initErr)
	}
	return nil
}

func (r *YARNResourceSyncReconciler) getYARNClient(yarnNode *yarnNode) (*yarnclient.YarnClient, error) {
	if yarnNode.ClusterID == "" && r.yarnClient != nil {
		return r.yarnClient, nil
	} else if yarnNode.ClusterID == "" && r.yarnClient == nil {
		yarnClient, err := yarnclient.CreateYarnClient()
		if err != nil {
			return nil, err
		}
		r.yarnClient = yarnClient
		return yarnClient, nil
	}

	//yarnNode.ClusterID != ""
	if clusterClient, exist := r.yarnClients[yarnNode.ClusterID]; exist {
		return clusterClient, nil
	}
	// create new client by cluster id
	clusterClient, err := yarnclient.CreateYarnClientByClusterID(yarnNode.ClusterID)
	if err != nil {
		return nil, err
	}
	r.yarnClients[yarnNode.ClusterID] = clusterClient
	return clusterClient, nil
}
