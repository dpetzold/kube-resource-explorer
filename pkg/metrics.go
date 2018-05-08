package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

type MetricType string

const (
	MEM = MetricType("mem")
	CPU = MetricType("cpu")
)

type ContainerMetrics struct {
	ContainerName string
	PodName       string
	NodeName      string

	MetricType MetricType
	Min        *resource.Quantity
	Max        *resource.Quantity
	Avg        *resource.Quantity
	Mode       *resource.Quantity
	Last       *resource.Quantity
	DataPoints int64
}

func QuantityStr(quantity *resource.Quantity, unit string) string {
	switch unit {
	case "m":
		return fmt.Sprintf("%vm", quantity.MilliValue())
	case "Mi":
		return fmt.Sprintf("%vMi", quantity.Value()/(1024*1024))
	default:
		return quantity.String()
	}
}

func (m *ContainerMetrics) fmtCpu() []string {
	return []string{
		QuantityStr(m.Last, "m"),
		QuantityStr(m.Min, "m"),
		QuantityStr(m.Max, "m"),
		QuantityStr(m.Avg, "m"),
	}
}

func (m *ContainerMetrics) fmtMem() []string {
	return []string{
		QuantityStr(m.Last, "Mi"),
		QuantityStr(m.Min, "Mi"),
		QuantityStr(m.Max, "Mi"),
		QuantityStr(m.Mode, "Mi"),
	}
}

func (m *ContainerMetrics) toSlice() []string {
	switch m.MetricType {
	case MEM:
		return m.fmtMem()
	case CPU:
		return m.fmtCpu()
	}

	return nil
}

func GroupMetricsBy(metrics []*ContainerMetrics, fields ...string) map[string]map[string]map[string]*ContainerMetrics {
	grouping := make(map[string]map[string]map[string]*ContainerMetrics)

	for _, m := range metrics {
		if n, ok := grouping[m.NodeName]; ok {
			if p, ok := n[m.PodName]; ok {
				p[m.ContainerName] = m
			} else {
				d := make(map[string]*ContainerMetrics)
				n[m.PodName] = d
				n[m.PodName][m.ContainerName] = m
			}
		} else {
			d2 := make(map[string]map[string]*ContainerMetrics)
			grouping[m.NodeName] = d2
			d := make(map[string]*ContainerMetrics)
			grouping[m.NodeName][m.PodName] = d
			grouping[m.NodeName][m.PodName][m.ContainerName] = m
		}
	}
	return grouping
}
