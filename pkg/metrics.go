package main

import (
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	log "github.com/Sirupsen/logrus"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
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

func evaluateMemMetrics(it *monitoring.TimeSeriesIterator) *ContainerMetrics {

	var points []*monitoringpb.Point
	set := make(map[int64]int)

	for {
		resp, err := it.Next()
		// This doesn't work
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.WithError(err).Debug("iterating")
			break
		}

		log.Debug(resp.Metric)
		log.Debug(resp.Resource)

		for _, point := range resp.Points {
			value := int64(point.Value.GetInt64Value())
			if _, ok := set[value]; ok {
				set[value] += 1
			} else {
				set[value] = 1
			}
			points = append(points, point)
		}
	}

	var data []int64
	for k, _ := range set {
		data = append(data, k)
	}

	sortPointsAsc(points)

	min, max := MinMax_int64(data)
	format := resource.BinarySI
	return &ContainerMetrics{
		MetricType: MEM,
		Last:       resource.NewQuantity(int64(points[0].Value.GetInt64Value()), format),
		Min:        resource.NewQuantity(min, format),
		Max:        resource.NewQuantity(max, format),
		Mode:       resource.NewQuantity(mode_int64(set), format),
		DataPoints: int64(len(points)),
	}
}

func evaluateCpuMetrics(it *monitoring.TimeSeriesIterator) *ContainerMetrics {
	var points []*monitoringpb.Point

	for {
		resp, err := it.Next()
		// This doesn't work
		if err == iterator.Done {
			break
		}
		if err != nil {
			// probably isn't a critical error, see above
			log.WithError(err).Debug("iterating")
			break
		}

		log.Debug(resp.Metric)
		log.Debug(resp.Resource)

		for _, point := range resp.Points {
			points = append(points, point)
		}
	}

	sortPointsAsc(points)

	var data []int64

	for i := 1; i < len(points); i++ {
		cur := points[i]
		prev := points[i-1]

		interval := cur.Interval.EndTime.Seconds - prev.Interval.EndTime.Seconds

		delta := float64(cur.Value.GetDoubleValue()) - float64(prev.Value.GetDoubleValue())
		data = append(data, int64((delta/float64(interval))*1000))
	}

	min, max := MinMax_int64(data)

	format := resource.DecimalSI
	return &ContainerMetrics{
		MetricType: CPU,
		Last:       resource.NewMilliQuantity(data[0], format),
		Min:        resource.NewMilliQuantity(min, format),
		Max:        resource.NewMilliQuantity(max, format),
		Avg:        resource.NewMilliQuantity(int64(average_int64(data)), format),
		DataPoints: int64(len(points)),
	}
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
