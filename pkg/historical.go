package main

import (
	// Imports the Stackdriver Monitoring client package.

	"fmt"
	"time"

	api_v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

func (c *StackDriverClient) getPodMetrics(pod api_v1.Pod, duration time.Duration, metric_type MetricType) []*ContainerMetrics {

	var metrics []*ContainerMetrics

	for _, container := range pod.Spec.Containers {
		m := c.getContainerMetrics(container.Name, pod.ObjectMeta.UID, duration, metric_type)
		m.PodName = pod.GetName()
		metrics = append(metrics, m)
	}

	return metrics
}

func (c *StackDriverClient) getNodeMetrics(node api_v1.Node, namespace string, duration time.Duration, metric_type MetricType) []*ContainerMetrics {

	fieldSelector, err := fields.ParseSelector(
		fmt.Sprintf("spec.nodeName=%s,status.phase!=%s,status.phase!=%s",
			node.GetName(), string(api_v1.PodSucceeded), string(api_v1.PodFailed)),
	)
	if err != nil {
		panic(err.Error())
	}

	nodeNonTerminatedPodsList, err := c.clientset.Core().Pods(
		namespace,
	).List(
		metav1.ListOptions{FieldSelector: fieldSelector.String()},
	)
	if err != nil {
		panic(err.Error())
	}

	var nodeMetrics []*ContainerMetrics

	for _, pod := range nodeNonTerminatedPodsList.Items {
		metrics := c.getPodMetrics(pod, duration, metric_type)
		for _, m := range metrics {
			m.NodeName = node.GetName()
		}
		nodeMetrics = append(nodeMetrics, metrics...)
	}

	return nodeMetrics
}

func (c *StackDriverClient) getMetrics(namespace string, duration time.Duration, metric_type MetricType) []*ContainerMetrics {

	nodes, err := c.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var metrics []*ContainerMetrics

	for _, node := range nodes.Items {
		m := c.getNodeMetrics(node, namespace, duration, metric_type)
		metrics = append(metrics, m...)
	}

	return metrics
}
