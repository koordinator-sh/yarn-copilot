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
	"math"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cri-api/pkg/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn"
	yarnserver "github.com/koordinator-sh/goyarn/pkg/yarn/apis/proto/hadoopyarn/server"
	"github.com/koordinator-sh/goyarn/pkg/yarn/cache"
	yarnclient "github.com/koordinator-sh/goyarn/pkg/yarn/client"
)

const (
	Name          = "yarnresource"
	YarnNamespace = "yarn"
)

type YARNResourceSyncReconciler struct {
	client.Client
	yarnClient  *yarnclient.YarnClient
	yarnClients map[string]*yarnclient.YarnClient
	cache       *cache.Nodes
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
		return ctrl.Result{}, r.updateYarnAllocatedResource(node, 0, 0)
	}

	// TODO exclude batch pod requested
	batchCPU, batchMemory, err := r.GetNodeOfflineResource(node, yarnNode)
	if err != nil {
		return ctrl.Result{}, err
	}
	klog.V(4).Infof("get node offline resource cpu: %d, memory: %d, name: %s", batchCPU.Value(), batchMemory.Value(), node.Name)

	vcores, memoryMB := resourceReserved(batchCPU.ScaledValue(resource.Kilo), batchMemory.ScaledValue(resource.Mega))

	// TODO control update frequency
	if err := r.updateYARNNodeResource(yarnNode, vcores, memoryMB); err != nil {
		klog.Warningf("update batch resource to yarn node %+v failed, k8s node name: %s, error %v", yarnNode, node.Name, err)
		return ctrl.Result{Requeue: true}, err
	}
	klog.V(4).Infof("update batch resource to yarn node %+v finish, cpu-core %v, memory-mb %v, k8s node name: %s",
		yarnNode, vcores, memoryMB, node.Name)

	core, mb, err := r.getYARNNodeAllocatedResource(yarnNode)
	if err != nil {
		return reconcile.Result{}, err
	}
	if err := r.updateYarnAllocatedResource(node, core, mb); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *YARNResourceSyncReconciler) GetNodeOfflineResource(node *corev1.Node, yarnNode *cache.YarnNode) (batchCPU resource.Quantity, batchMemory resource.Quantity, err error) {
	batchCPU, cpuExist := node.Status.Allocatable[BatchCPU]
	batchMemory, memExist := node.Status.Allocatable[BatchMemory]
	if !cpuExist {
		batchCPU = *resource.NewQuantity(0, resource.DecimalSI)
	}
	if !memExist {
		batchMemory = *resource.NewQuantity(0, resource.BinarySI)
	}
	if node.Annotations == nil || len(node.Annotations[actualOfflineResourceAnnotationKey]) == 0 {
		return
	}
	//cpu, mem, err := r.getYARNNodeAllocatedResource(yarnNode)
	//if err != nil {
	//	return
	//}
	//batchCPU.Add(*resource.NewQuantity(int64(cpu*1000), resource.DecimalSI))
	//batchMemory.Add(*resource.NewQuantity(int64(mem*1024*1024), resource.BinarySI))
	var actualResource corev1.ResourceList
	err = json.Unmarshal([]byte(node.Annotations[actualOfflineResourceAnnotationKey]), &actualResource)
	if err != nil {
		return
	}
	batchCPU, cpuExist = actualResource[BatchCPU]
	batchMemory, memExist = actualResource[BatchMemory]
	if !cpuExist {
		batchCPU = *resource.NewQuantity(0, resource.DecimalSI)
	}
	if !memExist {
		batchMemory = *resource.NewQuantity(0, resource.BinarySI)
	}
	return
}

type ResourceInfo map[string]resource.Quantity

type AllocatedResource map[string]*ResourceInfo

func (r *YARNResourceSyncReconciler) updateYarnAllocatedResource(node *corev1.Node, vcores int32, memoryMB int64) error {
	allocatedResource, err := r.GetAllocatedResource(node)
	if err != nil {
		return err
	}
	cpu := *resource.NewQuantity(int64(vcores), resource.DecimalSI)
	memory := *resource.NewQuantity(memoryMB*1024*1024, resource.BinarySI)
	allocatedResource["yarnAllocated"] = &ResourceInfo{
		string(BatchCPU):    cpu,
		string(BatchMemory): memory,
	}
	return r.SetAllocatedResource(node, allocatedResource)
}

func (r *YARNResourceSyncReconciler) SetAllocatedResource(node *corev1.Node, resource AllocatedResource) error {
	marshal, err := json.Marshal(resource)
	if err != nil {
		return err
	}
	newNode := node.DeepCopy()
	newNode.Annotations[yarnNodeAllocatedResourceAnnotationKey] = string(marshal)
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
	klog.Infof("update node %s", node.Name)
	return r.Client.Patch(context.TODO(), newNode, client.RawPatch(types.StrategicMergePatchType, patchBytes))
}

func (r *YARNResourceSyncReconciler) GetAllocatedResource(node *corev1.Node) (AllocatedResource, error) {
	if node.GetAnnotations() == nil || node.GetAnnotations()[yarnNodeAllocatedResourceAnnotationKey] == "" {
		return map[string]*ResourceInfo{}, nil
	}
	var res map[string]*ResourceInfo
	return res, json.Unmarshal([]byte(node.GetAnnotations()[yarnNodeAllocatedResourceAnnotationKey]), &res)
}

func Add(mgr ctrl.Manager) error {

	clients, err := yarnclient.GetAllKnownClients()
	if err != nil {
		return err
	}
	nodeCache := cache.NewNodes(clients)
	go nodeCache.Sync()
	coll := NewYarn(nodeCache)
	if _ = metrics.Registry.Register(coll); err != nil {
		return err
	}
	r := &YARNResourceSyncReconciler{
		Client:      mgr.GetClient(),
		yarnClients: clients,
		cache:       nodeCache,
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
		klog.Warningf("get %v node manager pod on node %v, will select the first one", len(podList.Items), node.Name)
		return &podList.Items[0], nil
	}
	return &podList.Items[0], nil
}

// CPU暂时不考虑预留，内存预留5% 给nodemanager，且保证汇报资源量不大于100GB【业务方需求，调度任务过多云盘IO压力过大】
func resourceReserved(vcore, memoryMB int64) (int32, int64) {
	klog.V(3).Infof("before reserved %d %d", vcore, memoryMB)
	// convert to yarn format.
	memMB := int64(math.Floor(float64(memoryMB) * float64(0.95)))
	memMBMax := int64(160 * 1024) // 大于160GB，默认只上报160GB
	memMBMin := int64(10 * 1024)  // 小于10GB，默认不上报，禁止离线调度上来
	if memMB >= memMBMax {
		memMB = memMBMax
	}
	if memMB < memMBMin {
		memMB = int64(0)
	}
	return int32(vcore), memMB
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
		return nil, fmt.Errorf("yarn nm id %v not exist in node annotationv", YarnNodeIdAnnotation)
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
	if clusterID, exist := nmPod.Annotations[YarnClusterIDAnnotation]; exist {
		yarnNode.ClusterID = clusterID
	}
	return yarnNode, nil
}

func (r *YARNResourceSyncReconciler) updateYARNNodeResource(yarnNode *cache.YarnNode, vcores int32, memoryMB int64) error {

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

func (r *YARNResourceSyncReconciler) getYARNClient(yarnNode *cache.YarnNode) (*yarnclient.YarnClient, error) {
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

func (r *YARNResourceSyncReconciler) getYARNNodeAllocatedResource(yarnNode *cache.YarnNode) (vcores int32, memoryMB int64, err error) {
	nodeResource, exist := r.cache.GetNodeResource(yarnNode)
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
