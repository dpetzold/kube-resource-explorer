package main

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubectl/metricsutil"
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

func (k *KubeClient) ActivePods(namespace, nodeName string) ([]api_v1.Pod, error) {

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

func NodeCapacity(node *api_v1.Node) api_v1.ResourceList {
	allocatable := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		allocatable = node.Status.Allocatable
	}
	return allocatable
}

// Return NodeResources struct for the specified object
func (k *KubeClient) NodeResourceUsage(node *api_v1.Node) (*ResourceUsage, error) {

	client := metricsutil.DefaultHeapsterMetricsClient(k.clientset.Core())

	metricsList, err := client.GetNodeMetrics(node.GetName(), labels.Everything().String())
	if err != nil {
		return nil, err
	}

	if len(metricsList.Items) != 1 {
		return nil, fmt.Errorf("Got bad number of results from client.GetNodeMetrics")
	}

	metrics := metricsList.Items[0]

	capacity := NodeCapacity(node)

	cpuQuantity := metrics.Usage[api_v1.ResourceCPU]
	memoryQuantity := metrics.Usage[api_v1.ResourceMemory]

	cpuUsage := NewCpuResource(cpuQuantity.MilliValue())
	memoryUsage := NewMemoryResource(memoryQuantity.Value())

	percentCpu := cpuUsage.calcPercentage(capacity.Cpu())
	percentMemory := memoryUsage.calcPercentage(capacity.Memory())

	return &ResourceUsage{
		Name:          node.GetName(),
		CpuUsage:      cpuUsage,
		PercentCpu:    percentCpu,
		MemoryUsage:   memoryUsage,
		PercentMemory: percentMemory,
	}, nil
}

func (k *KubeClient) PodResourceUsage(namespace string) ([]*ResourceUsage, error) {

	client := metricsutil.DefaultHeapsterMetricsClient(k.clientset.Core())

	metricsList, err := client.GetPodMetrics("", "", true, labels.Everything())
	if err != nil {
		return nil, err
	}

	var resources []*ResourceUsage

	for _, item := range metricsList.Items {

		for _, metrics := range item.Containers {

			cpuQuantity := metrics.Usage[api_v1.ResourceCPU]
			memoryQuantity := metrics.Usage[api_v1.ResourceMemory]

			cpuUsage := NewCpuResource(cpuQuantity.MilliValue())
			memoryUsage := NewMemoryResource(memoryQuantity.Value())

			resources = append(resources, &ResourceUsage{
				Name:        fmt.Sprintf("%s/%s", item.ObjectMeta.Name, metrics.Name),
				CpuUsage:    cpuUsage,
				MemoryUsage: memoryUsage,
			})
		}

	}

	return resources, nil
}

// Return a list of container resources for all containers running on the specified node
func (k *KubeClient) NodeContainerResources(namespace, nodeName string) (resources []*ContainerResources, err error) {

	mc := k.clientset.Core().Nodes()
	node, err := mc.Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	activePodsList, err := k.ActivePods(namespace, nodeName)
	if err != nil {
		return nil, err
	}

	capacity := NodeCapacity(node)

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

// Return the resources in use by containers in the cluster as list of ContainerResources
func (k *KubeClient) ContainerResources(namespace string) (resources []*ContainerResources, err error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes.Items {
		nodeUsage, err := k.NodeContainerResources(namespace, node.GetName())
		if err != nil {
			return nil, err
		}
		resources = append(resources, nodeUsage...)
	}

	return resources, nil
}

func (k *KubeClient) Nodes() ([]api_v1.Node, error) {
	nodeList, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return nodeList.Items, nil
}

// Return the total cluster capacity as a ResourceList
func (k *KubeClient) ClusterCapacity() (capacity api_v1.ResourceList, err error) {

	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	capacity = api_v1.ResourceList{}

	for _, node := range nodes.Items {

		allocatable := NodeCapacity(&node)

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

// Driver for the resource usage command
func (k *KubeClient) ResourceUsage(namespace, sort string, reverse bool, csv bool) {

	resources, err := k.ContainerResources(namespace)
	if err != nil {
		panic(err.Error())
	}

	capacity, err := k.ClusterCapacity()
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

func (k *KubeClient) Events(namespace string) ([]api_v1.Event, error) {

	events, err := k.clientset.Core().Events(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return events.Items, nil
}
