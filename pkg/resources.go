package main

import "k8s.io/apimachinery/pkg/api/resource"

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

func (r ContainerResources) Validate(field string) bool {
	for _, v := range getFields(&r) {
		if field == v {
			return true
		}
	}
	return false
}
