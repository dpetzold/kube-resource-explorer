package main

import (
	monitoring "cloud.google.com/go/monitoring/apiv3"
	log "github.com/Sirupsen/logrus"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
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
	return &ContainerMetrics{
		MetricType: v1.ResourceMemory,
		MemoryLast: NewMemoryResource(points[0].Value.GetInt64Value()),
		MemoryMin:  NewMemoryResource(min),
		MemoryMax:  NewMemoryResource(max),
		MemoryMode: NewMemoryResource(mode_int64(set)),
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

	return &ContainerMetrics{
		MetricType: v1.ResourceCPU,
		CpuLast:    NewCpuResource(data[0]),
		CpuMin:     NewCpuResource(min),
		CpuMax:     NewCpuResource(max),
		CpuAvg:     NewCpuResource(int64(average_int64(data))),
		DataPoints: int64(len(points)),
	}
}
