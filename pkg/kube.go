package main

import (
	"fmt"
	"time"

	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

func containerRequestsAndLimits(container *api_v1.Container) (reqs map[api_v1.ResourceName]resource.Quantity, limits map[api_v1.ResourceName]resource.Quantity) {
	reqs, limits = map[api_v1.ResourceName]resource.Quantity{}, map[api_v1.ResourceName]resource.Quantity{}

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

func (k *KubeClient) listNodeResources(name string, namespace string) ([]*ResourceAllocation, error) {
	mc := k.clientset.Core().Nodes()
	node, err := mc.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	fieldSelector, err := fields.ParseSelector(
		fmt.Sprintf("spec.nodeName=%s,status.phase!=%s,status.phase!=%s",
			name, string(api_v1.PodSucceeded), string(api_v1.PodFailed)),
	)
	if err != nil {
		return nil, err
	}

	nodeNonTerminatedPodsList, err := k.clientset.Core().Pods(
		namespace,
	).List(
		metav1.ListOptions{FieldSelector: fieldSelector.String()},
	)
	if err != nil {
		if !errors.IsForbidden(err) {
			return nil, err
		}
	}

	allocatable := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		allocatable = node.Status.Allocatable
	}

	var resourceAllocation []*ResourceAllocation

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/describe.go#L2970
	for _, pod := range nodeNonTerminatedPodsList.Items {
		for _, container := range pod.Spec.Containers {
			req, limit := containerRequestsAndLimits(&container)
			cpuReq, cpuLimit, memoryReq, memoryLimit := req[api_v1.ResourceCPU], limit[api_v1.ResourceCPU], req[api_v1.ResourceMemory], limit[api_v1.ResourceMemory]
			percentCpuReq := float64(cpuReq.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
			percentCpuLimit := float64(cpuLimit.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
			percentMemoryReq := float64(memoryReq.Value()) / float64(allocatable.Memory().Value()) * 100
			percentMemoryLimit := float64(memoryLimit.Value()) / float64(allocatable.Memory().Value()) * 100

			resourceAllocation = append(resourceAllocation, &ResourceAllocation{
				Name:               fmt.Sprintf("%s/%s", pod.GetName(), container.Name),
				Namespace:          pod.GetNamespace(),
				CpuReq:             &cpuReq,
				CpuLimit:           &cpuLimit,
				PercentCpuReq:      int64(percentCpuReq),
				PercentCpuLimit:    int64(percentCpuLimit),
				MemReq:             &memoryReq,
				MemLimit:           &memoryLimit,
				PercentMemoryReq:   int64(percentMemoryReq),
				PercentMemoryLimit: int64(percentMemoryLimit),
			})
		}
	}

	return resourceAllocation, nil
}

func (k *KubeClient) ListResources(namespace string) ([]*ResourceAllocation, error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resourceAllocation []*ResourceAllocation

	for _, node := range nodes.Items {
		nodeUsage, err := k.listNodeResources(node.GetName(), namespace)
		if err != nil {
			return nil, err
		}
		resourceAllocation = append(resourceAllocation, nodeUsage...)
	}

	return resourceAllocation, nil
}

func (k *KubeClient) TotalResources(namespace string) ([]*ResourceAllocation, error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resourceAllocation []*ResourceAllocation

	for _, node := range nodes.Items {
		nodeUsage, err := k.listNodeResources(node.GetName(), namespace)
		if err != nil {
			return nil, err
		}
		resourceAllocation = append(resourceAllocation, nodeUsage...)
	}

	return resourceAllocation, nil
}

func (k *KubeClient) getMetrics(s *StackDriverClient, namespace string, duration time.Duration, metric_type MetricType) []*ContainerMetrics {

	fieldSelector, err := fields.ParseSelector(
		fmt.Sprintf("status.phase!=%s,status.phase!=%s", string(api_v1.PodSucceeded), string(api_v1.PodFailed)),
	)
	if err != nil {
		panic(err.Error())
	}

	nodeNonTerminatedPodsList, err := k.clientset.Core().Pods(
		namespace,
	).List(
		metav1.ListOptions{FieldSelector: fieldSelector.String()},
	)
	if err != nil {
		panic(err.Error())
	}

	var metrics []*ContainerMetrics

	for _, pod := range nodeNonTerminatedPodsList.Items {

		for _, container := range pod.Spec.Containers {
			m := s.getContainerMetrics(container.Name, pod.ObjectMeta.UID, duration, metric_type)
			m.PodName = pod.GetName()
			metrics = append(metrics, m)
		}
	}

	return metrics
}
