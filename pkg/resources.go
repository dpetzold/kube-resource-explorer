package main

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ContainerResources struct {
	Name               string
	Namespace          string
	CpuReq             *resource.Quantity
	CpuLimit           *resource.Quantity
	PercentCpuReq      int64
	PercentCpuLimit    int64
	MemReq             *resource.Quantity
	MemLimit           *resource.Quantity
	PercentMemoryReq   int64
	PercentMemoryLimit int64
}

func calcPercentage(dividend, divisor int64) int64 {
	return int64(float64(dividend) / float64(divisor) * 100)
}

func calcCpuPercentage(dividend resource.Quantity, divisor v1.ResourceList) int64 {
	return calcPercentage(dividend.MilliValue(), divisor.Cpu().MilliValue())
}

func calcMemoryPercentage(dividend resource.Quantity, divisor v1.ResourceList) int64 {
	return calcPercentage(dividend.Value(), divisor.Memory().Value())
}

func (r ContainerResources) Validate(field string) bool {
	for _, v := range getFields(&r) {
		if field == v {
			return true
		}
	}
	return false
}
