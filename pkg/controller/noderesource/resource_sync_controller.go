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
	"k8s.io/apimachinery/pkg/types"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cri-api/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	yarnmetrics "github.com/koordinator-sh/goyarn/pkg/controller/metrics"
	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	yarnserver "github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn/server"
	"github.com/koordinator-sh/goyarn/pkg/yarn/cache"
	yarnclient "github.com/koordinator-sh/goyarn/pkg/yarn/client"
)

const (
	Name = "yarnresource"
)

type YARNResourceSyncReconciler struct {
	client.Client
	yarnClient    yarnclient.YarnClient
	yarnClients   map[string]yarnclient.YarnClient
	yarnNodeCache *cache.NodesSyncer
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
		klog.V(3).Infof("yarn node not exist on node %v, clear yarn allocated resource, detail %+v", req.Name, yarnNode)
		if err := r.updateYARNAllocatedResource(node, 0, 0); err != nil {
			klog.Warningf("failed to clear yarn allocated resource for node %v", req.Name)
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// TODO exclude batch pod requested
	batchCPU, batchMemory, err := getNodeBatchResource(node)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	klog.V(4).Infof("get node batch resource cpu: %d, memory: %d, name: %s", batchCPU.Value(), batchMemory.Value(), node.Name)

	vcores, memoryMB := calculate(batchCPU, batchMemory)

	// TODO control update frequency by ignore unnecessary node update event
	if err := r.updateYARNNodeResource(yarnNode, vcores, memoryMB); err != nil {
		klog.Warningf("update batch resource to yarn node %+v failed, k8s node name: %s, error %v", yarnNode, node.Name, err)
		return ctrl.Result{Requeue: true}, err
	}
	klog.V(4).Infof("update batch resource to yarn node %+v finish, cpu-core %v, memory-mb %v, k8s node name: %s",
		yarnNode, vcores, memoryMB, node.Name)

	core, mb, err := r.getYARNNodeAllocatedResource(yarnNode)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	if err := r.updateYARNAllocatedResource(node, core, mb); err != nil {
		klog.Warningf("failed to update yarn allocated resource for node %v, error %v", node.Name, err)
		return reconcile.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func getNodeBatchResource(node *corev1.Node) (batchCPU resource.Quantity, batchMemory resource.Quantity, err error) {
	if node == nil {
		return
	}

	batchCPU, cpuExist := node.Status.Allocatable[BatchCPU]
	batchMemory, memExist := node.Status.Allocatable[BatchMemory]
	if !cpuExist {
		batchCPU = *resource.NewQuantity(0, resource.DecimalSI)
	}
	if !memExist {
		batchMemory = *resource.NewQuantity(0, resource.BinarySI)
	}
	originAllocatableRes, err := GetOriginExtendedAllocatableRes(node.Annotations)
	if err == nil && originAllocatableRes == nil {
		// koordiantor <= 1.3.0, use node status as origin batch total
		return
	} else if err != nil {
		klog.Warningf("get origin allocatable from node %v annotation failed, error: %v", node.Name, err)
		return
	}
	batchCPU, cpuExist = originAllocatableRes[BatchCPU]
	batchMemory, memExist = originAllocatableRes[BatchMemory]
	if !cpuExist {
		batchCPU = *resource.NewQuantity(0, resource.DecimalSI)
	}
	if !memExist {
		batchMemory = *resource.NewQuantity(0, resource.BinarySI)
	}
	return
}

func (r *YARNResourceSyncReconciler) updateYARNAllocatedResource(node *corev1.Node, vcores int32, memoryMB int64) error {
	if node == nil {
		return nil
	}
	newNode := node.DeepCopy()
	if err := SetYARNAllocatedResource(newNode.Annotations, vcores, memoryMB); err != nil {
		return err
	}

	oldData, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal the existing node %#v: %v", node, err)
	}
	newData, err := json.Marshal(newNode)
	if err != nil {
		return fmt.Errorf("failed to marshal the new node %#v: %v", newNode, err)
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &corev1.Node{})
	if err != nil {
		return fmt.Errorf("failed to create a two-way merge patch: %v", err)
	}
	klog.V(4).Infof("update node %s with yarn allocated cpu: %v, memory: %v, patch %v", node.Name, vcores, memoryMB, string(patchBytes))
	return r.Client.Patch(context.TODO(), node, client.RawPatch(types.StrategicMergePatchType, patchBytes))
}

func Add(mgr ctrl.Manager) error {
	clients, err := yarnclient.DefaultYarnClientFactory.CreateAllYarnClients()
	if err != nil {
		return err
	}
	yarnNodesSyncer := cache.NewNodesSyncer(clients)

	coll := yarnmetrics.NewYarnMetricCollector(yarnNodesSyncer)
	if err = metrics.Registry.Register(coll); err != nil {
		return err
	}
	r := &YARNResourceSyncReconciler{
		Client:        mgr.GetClient(),
		yarnClients:   clients,
		yarnNodeCache: yarnNodesSyncer,
	}

	if err := mgr.Add(yarnNodesSyncer); err != nil {
		return err
	}
	return r.SetupWithManager(mgr)
}

func (r *YARNResourceSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// TODO use source.Channel to handle yarn node requested update event
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named(Name).
		Complete(r)
}

func (r *YARNResourceSyncReconciler) getYARNNodeManagerPod(node *corev1.Node) (*corev1.Pod, error) {
	if node == nil {
		return nil, nil
	}
	opts := []client.ListOption{
		client.MatchingLabels{YarnNMComponentLabel: YarnNMComponentValue},
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
		klog.Warningf("get %v node manager pod on node %v, will select the first one %v/%v", len(podList.Items),
			node.Name, podList.Items[0].Namespace, podList.Items[0].Name)
	}
	return &podList.Items[0], nil
}

func (r *YARNResourceSyncReconciler) getYARNNode(node *corev1.Node) (*cache.YarnNode, error) {
	if node == nil {
		return nil, nil
	}

	nmPod, err := r.getYARNNodeManagerPod(node)
	if err != nil {
		return nil, err
	} else if nmPod == nil {
		return nil, nil
	}

	podAnnoNodeId, exists := nmPod.Annotations[YarnNodeIdAnnotation]
	if !exists {
		return nil, fmt.Errorf("yarn nm id %v not exist in node annotation", YarnNodeIdAnnotation)
	}
	tokens := strings.Split(podAnnoNodeId, ":")
	if len(tokens) != 2 {
		return nil, fmt.Errorf("yarn nm id %v format is illegal", podAnnoNodeId)
	}
	port, err := strconv.Atoi(tokens[1])
	if err != nil {
		return nil, fmt.Errorf("yarn nm id port %v format is illegal", podAnnoNodeId)
	}

	yarnNode := &cache.YarnNode{
		Name: tokens[0],
		Port: int32(port),
	}
	if clusterID, exist := nmPod.Annotations[PodYarnClusterIDAnnotationKey]; exist {
		yarnNode.ClusterID = clusterID
	} else {
		yarnNode.ClusterID = yarnclient.DefaultClusterID
	}
	return yarnNode, nil
}

func (r *YARNResourceSyncReconciler) updateYARNNodeResource(yarnNode *cache.YarnNode, vcores, memoryMB int64) error {
	if yarnNode == nil {
		return nil
	}
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
						VirtualCores: pointer.Int32(int32(vcores)),
					},
				},
			},
		},
	}
	yarnClient, err := r.getYARNClient(yarnNode)
	if err != nil || yarnClient == nil {
		return err
	}
	if resp, err := yarnClient.UpdateNodeResource(request); err != nil {
		initErr := yarnClient.Reinitialize()
		return fmt.Errorf("UpdateNodeResource resp %v, error %v, reinitialize error %v", resp, err, initErr)
	}
	return nil
}

func (r *YARNResourceSyncReconciler) getYARNClient(yarnNode *cache.YarnNode) (yarnclient.YarnClient, error) {
	if yarnNode == nil {
		return nil, nil
	} else if yarnNode.ClusterID == "" && r.yarnClient != nil {
		return r.yarnClient, nil
	} else if yarnNode.ClusterID == "" && r.yarnClient == nil {
		yarnClient, err := yarnclient.DefaultYarnClientFactory.CreateDefaultYarnClient()
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
	clusterClient, err := yarnclient.DefaultYarnClientFactory.CreateYarnClientByClusterID(yarnNode.ClusterID)
	if err != nil {
		return nil, err
	}
	if r.yarnClients == nil {
		r.yarnClients = map[string]yarnclient.YarnClient{}
	}
	r.yarnClients[yarnNode.ClusterID] = clusterClient
	return clusterClient, nil
}

func (r *YARNResourceSyncReconciler) getYARNNodeAllocatedResource(yarnNode *cache.YarnNode) (vcores int32, memoryMB int64, err error) {
	if yarnNode == nil {
		return 0, 0, nil
	}
	nodeResource, exist := r.yarnNodeCache.GetNodeResource(yarnNode)
	if !exist {
		return 0, 0, nil
	}
	if nodeResource.Used.VirtualCores != nil {
		vcores = *nodeResource.Used.VirtualCores
	}
	if nodeResource.Used.Memory != nil {
		memoryMB = *nodeResource.Used.Memory
	}
	return
}
