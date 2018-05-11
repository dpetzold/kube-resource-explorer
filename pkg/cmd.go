package main

import (
	"time"

	"k8s.io/api/core/v1"
)

func (k *KubeClient) historical(project, namespace string, resourceName v1.ResourceName, duration time.Duration, sort string, reverse bool) {

	stackDriver := NewStackDriverClient(project)

	activePods, err := k.getActivePods(namespace, "")
	if err != nil {
		panic(err.Error())
	}

	metrics := stackDriver.getMetrics(activePods, duration, resourceName)
	PrintContainerMetrics(metrics, resourceName, duration, sort, reverse)
}

func (k *KubeClient) resourceUsage(namespace, sort string, reverse bool) {
	resources, err := k.GetContainerResources(namespace)
	if err != nil {
		panic(err.Error())
	}

	capacity, err := k.GetClusterCapacity()
	if err != nil {
		panic(err.Error())
	}

	PrintResourceUsage(capacity, resources, sort, reverse)
}
