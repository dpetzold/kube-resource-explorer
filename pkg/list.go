package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	api_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	resourcehelper "k8s.io/kubernetes/pkg/api/v1/resource"

	"github.com/ryanuber/columnize"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type ResourceLister struct {
	clientset *kubernetes.Clientset
}

type ResourceUsage struct {
	Name                string
	Namespace           string
	CpuReq              resource.Quantity
	CpuLimit            resource.Quantity
	FractionCpuReq      int64
	FractionCpuLimit    int64
	MemReq              resource.Quantity
	MemLimit            resource.Quantity
	FractionMemoryReq   int64
	FractionMemoryLimit int64
}

func (r *ResourceUsage) getFields() []string {
	var fields []string

	s := reflect.ValueOf(r).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		fields = append(fields, typeOfT.Field(i).Name)
	}
	return fields
}

func (r *ResourceUsage) getField(field string) interface{} {
	v := reflect.ValueOf(r)
	f := reflect.Indirect(v).FieldByName(field)
	return f.Interface()
}

func cmp(r []*ResourceUsage, field string, i, j int, reverse bool) bool {
	f1 := r[i].getField(field)
	f2 := r[j].getField(field)

	q1, ok := f1.(resource.Quantity)
	if ok {
		q2 := f2.(resource.Quantity)
		if reverse {
			return q1.Cmp(q2) < 0
		}
		return q1.Cmp(q2) > 0
	}

	v1, ok := f1.(int64)
	if ok {
		v2 := f2.(int64)
		if reverse {
			return v1 < v2
		}
		return v1 > v2

	}

	s1, ok := f1.(string)
	if ok {
		s2 := f2.(string)
		if reverse {
			return strings.Compare(s1, s2) > 0
		}
		return strings.Compare(s1, s2) < 0
	}

	panic("uknown type")
}

func NewResourceLister(
	clientset *kubernetes.Clientset,
) *ResourceLister {
	return &ResourceLister{
		clientset: clientset,
	}
}

func (r *ResourceLister) ListNodeResources(name string, namespace string) ([]*ResourceUsage, error) {
	mc := r.clientset.Core().Nodes()
	node, err := mc.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	fieldSelector, err := fields.ParseSelector(
		fmt.Sprintf("spec.nodeName=%s,status.phase!=%s,status.phase!=%s", name, string(api_v1.PodSucceeded), string(api_v1.PodFailed)),
	)
	if err != nil {
		return nil, err
	}
	// in a policy aware setting, users may have access to a node, but not all pods
	// in that case, we note that the user does not have access to the pods
	// canViewPods := true
	nodeNonTerminatedPodsList, err := r.clientset.Core().Pods(
		namespace,
	).List(
		metav1.ListOptions{FieldSelector: fieldSelector.String()},
	)
	if err != nil {
		if !errors.IsForbidden(err) {
			return nil, err
		}
		// canViewPods = false
	}

	allocatable := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		allocatable = node.Status.Allocatable
	}

	var resourceUsage []*ResourceUsage

	// https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/describe.go#L2970
	for _, pod := range nodeNonTerminatedPodsList.Items {
		req, limit := resourcehelper.PodRequestsAndLimits(&pod)
		cpuReq, cpuLimit, memoryReq, memoryLimit := req[api_v1.ResourceCPU], limit[api_v1.ResourceCPU], req[api_v1.ResourceMemory], limit[api_v1.ResourceMemory]
		fractionCpuReq := float64(cpuReq.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
		fractionCpuLimit := float64(cpuLimit.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
		fractionMemoryReq := float64(memoryReq.Value()) / float64(allocatable.Memory().Value()) * 100
		fractionMemoryLimit := float64(memoryLimit.Value()) / float64(allocatable.Memory().Value()) * 100

		resourceUsage = append(resourceUsage, &ResourceUsage{
			Name:                pod.GetName(),
			Namespace:           pod.GetNamespace(),
			CpuReq:              cpuReq,
			CpuLimit:            cpuLimit,
			FractionCpuReq:      int64(fractionCpuReq),
			FractionCpuLimit:    int64(fractionCpuLimit),
			MemReq:              memoryReq,
			MemLimit:            memoryLimit,
			FractionMemoryReq:   int64(fractionMemoryReq),
			FractionMemoryLimit: int64(fractionMemoryLimit),
		})
	}

	return resourceUsage, nil
}

func (r *ResourceLister) ListResources(namespace string, field string, reverse bool) error {
	nodes, err := r.clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var resourceUsage []*ResourceUsage

	for _, node := range nodes.Items {
		nodeUsage, _ := r.ListNodeResources(node.GetName(), namespace)
		resourceUsage = append(resourceUsage, nodeUsage...)
	}

	sort.Slice(resourceUsage, func(i, j int) bool {
		return cmp(resourceUsage, field, i, j, reverse)
	})

	rows := []string{
		"Namespace | Name | CpuReq | CpuReq% | CpuLimit | CpuLimit% | MemReq | MemReq% | MemLimit | MemLimit%",
		"--------- | ---- | ------ | ------- | -------- | --------- | ------ | ------- | -------- | ---------",
	}

	for _, u := range resourceUsage {
		row := strings.Join([]string{
			u.Namespace,
			u.Name,
			u.CpuReq.String(),
			fmt.Sprintf("%d%%", u.FractionCpuReq),
			u.CpuLimit.String(),
			fmt.Sprintf("%d%%", u.FractionCpuLimit),
			u.MemReq.String(),
			fmt.Sprintf("%d%%", u.FractionMemoryReq),
			u.MemLimit.String(),
			fmt.Sprintf("%d%%", u.FractionMemoryLimit),
		}, "| ")
		rows = append(rows, row)
	}

	fmt.Println(columnize.SimpleFormat(rows))

	return nil
}
