package main

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type KubeClient struct {
	clientset *kubernetes.Clientset
}

func NewKubeClient(
	clientset *kubernetes.Clientset,
) *KubeClient {
	return &KubeClient{
		clientset: clientset,
	}
}

func (k *KubeClient) getActivePods(namespace, nodeName string) ([]api_v1.Pod, error) {

	selector := fmt.Sprintf("status.phase!=%s,status.phase!=%s", string(api_v1.PodSucceeded), string(api_v1.PodFailed))
	if nodeName != "" {
		selector += fmt.Sprintf(",spec.nodeName=%s", nodeName)
	}

	fieldSelector, err := fields.ParseSelector(selector)
	if err != nil {
		return nil, err
	}

	activePods, err := k.clientset.Core().Pods(
		namespace,
	).List(
		metav1.ListOptions{FieldSelector: fieldSelector.String()},
	)
	if err != nil {
		return nil, err
	}

	return activePods.Items, err
}

func containerRequestsAndLimits(container *api_v1.Container) (reqs api_v1.ResourceList, limits api_v1.ResourceList) {
	reqs, limits = api_v1.ResourceList{}, api_v1.ResourceList{}

	for name, quantity := range container.Resources.Requests {
		if _, ok := reqs[name]; ok {
			panic(fmt.Sprintf("Duplicate key: %s", name))
		} else {
			reqs[name] = *quantity.Copy()
		}
	}

	for name, quantity := range container.Resources.Limits {
		if _, ok := limits[name]; ok {
			panic(fmt.Sprintf("Duplicate key: %s", name))
		} else {
			limits[name] = *quantity.Copy()
		}
	}
	return
}

func getNodeCapacity(node *api_v1.Node) api_v1.ResourceList {
	allocatable := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		allocatable = node.Status.Allocatable
	}
	return allocatable
}

func (k *KubeClient) getNodeResources(namespace, nodeName string) (resources []*ContainerResources, err error) {

	mc := k.clientset.Core().Nodes()
	node, err := mc.Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	activePodsList, err := k.getActivePods(namespace, nodeName)
	if err != nil {
		return nil, err
	}

	capacity := getNodeCapacity(node)

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/describe.go#L2970
	for _, pod := range activePodsList {
		for _, container := range pod.Spec.Containers {
			req, limit := containerRequestsAndLimits(&container)

			_cpuReq := req[api_v1.ResourceCPU]
			cpuReq := NewCpuResource(_cpuReq.MilliValue())

			_cpuLimit := limit[api_v1.ResourceCPU]
			cpuLimit := NewCpuResource(_cpuLimit.MilliValue())

			_memoryReq := req[api_v1.ResourceMemory]
			memoryReq := NewMemoryResource(_memoryReq.Value())

			_memoryLimit := limit[api_v1.ResourceMemory]
			memoryLimit := NewMemoryResource(_memoryLimit.Value())

			resources = append(resources, &ContainerResources{
				Name:               fmt.Sprintf("%s/%s", pod.GetName(), container.Name),
				Namespace:          pod.GetNamespace(),
				CpuReq:             cpuReq,
				CpuLimit:           cpuLimit,
				PercentCpuReq:      cpuReq.calcPercentage(capacity.Cpu()),
				PercentCpuLimit:    cpuLimit.calcPercentage(capacity.Cpu()),
				MemReq:             memoryReq,
				MemLimit:           memoryLimit,
				PercentMemoryReq:   memoryReq.calcPercentage(capacity.Memory()),
				PercentMemoryLimit: memoryLimit.calcPercentage(capacity.Memory()),
			})
		}
	}

	return resources, nil
}

func (k *KubeClient) GetContainerResources(namespace string) (resources []*ContainerResources, err error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		nodeUsage, err := k.getNodeResources(namespace, node.GetName())
		if err != nil {
			return nil, err
		}
		resources = append(resources, nodeUsage...)
	}

	return resources, nil
}

func (k *KubeClient) GetClusterCapacity() (capacity api_v1.ResourceList, err error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	capacity = api_v1.ResourceList{}

	for _, node := range nodes.Items {

		allocatable := getNodeCapacity(&node)

		for name, quantity := range allocatable {
			if value, ok := capacity[name]; ok {
				value.Add(quantity)
				capacity[name] = value
			} else {
				capacity[name] = *quantity.Copy()
			}
		}

	}

	return capacity, nil
}

func (k *KubeClient) resourceUsage(namespace, sort string, reverse bool, csv bool) {

	resources, err := k.GetContainerResources(namespace)
	if err != nil {
		panic(err.Error())
	}

	capacity, err := k.GetClusterCapacity()
	if err != nil {
		panic(err.Error())
	}

	rows := FormatResourceUsage(capacity, resources, sort, reverse)

	if csv {
		prefix := "kube-resource-usage"
		if namespace == "" {
			prefix += "-all"
		} else {
			prefix += fmt.Sprintf("-%s", namespace)
		}

		filename := ExportCSV(prefix, rows)
		fmt.Printf("Exported %d rows to %s\n", len(rows), filename)
	} else {
		PrintResourceUsage(rows)
	}
}
