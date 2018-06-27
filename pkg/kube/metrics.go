package kube

import (
	"k8s.io/api/core/v1"
)

type ContainerMetrics struct {
	ContainerName string
	PodName       string
	NodeName      string

	MetricType v1.ResourceName
	MemoryMin  *MemoryResource
	MemoryMax  *MemoryResource
	MemoryMode *MemoryResource
	MemoryLast *MemoryResource

	CpuMin  *CpuResource
	CpuMax  *CpuResource
	CpuAvg  *CpuResource
	CpuLast *CpuResource

	DataPoints int64
}

func (m ContainerMetrics) Validate(field string) bool {
	for _, v := range GetFields(&m) {
		if field == v {
			return true
		}
	}
	return false
}

func (m *ContainerMetrics) fmtCpu() []string {
	return []string{
		m.CpuLast.String(),
		m.CpuMin.String(),
		m.CpuMax.String(),
		m.CpuAvg.String(),
	}
}

func (m *ContainerMetrics) fmtMem() []string {
	return []string{
		m.MemoryLast.String(),
		m.MemoryMin.String(),
		m.MemoryMax.String(),
		m.MemoryMode.String(),
	}
}

func (m *ContainerMetrics) toSlice() []string {
	switch m.MetricType {
	case v1.ResourceMemory:
		return m.fmtMem()
	case v1.ResourceCPU:
		return m.fmtCpu()
	}

	return nil
}
