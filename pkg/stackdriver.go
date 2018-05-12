package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	log "github.com/Sirupsen/logrus"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type StackDriverClient struct {
	ctx     context.Context
	client  *monitoring.MetricClient
	project string
}

func NewStackDriverClient(project string) *StackDriverClient {
	ctx := context.Background()
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		panic(err.Error())
	}
	return &StackDriverClient{
		ctx:     ctx,
		client:  c,
		project: project,
	}
}

func sortPointsAsc(points []*monitoringpb.Point) {
	sort.Slice(points, func(i, j int) bool {
		return points[i].Interval.EndTime.Seconds > points[j].Interval.EndTime.Seconds
	})
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

func buildTimeSeriesFilter(m map[string]string) string {

	// buffer := make([]string, len(m))
	var buffer []string

	for k, v := range m {
		buffer = append(buffer, fmt.Sprintf("%s = \"%s\"", k, v))
	}

	return strings.Join(buffer, " AND ")
}

func (s *StackDriverClient) getTimeSeries(filter_map map[string]string, duration time.Duration) *monitoring.TimeSeriesIterator {

	filter := buildTimeSeriesFilter(filter_map)

	log.Debug(filter)

	end := time.Now().UTC()
	start := end.Add(-duration)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%s", s.project),
		Filter: filter,
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: start.Unix(),
				Nanos:   int32(start.Nanosecond()),
			},
			EndTime: &timestamp.Timestamp{
				Seconds: end.Unix(),
				Nanos:   int32(end.Nanosecond()),
			},
		},
	}

	return s.client.ListTimeSeries(s.ctx, req)
}

func (s *StackDriverClient) getContainerMetrics(container_name string, pod_uid types.UID, duration time.Duration, metric_type v1.ResourceName) *ContainerMetrics {

	var m *ContainerMetrics

	filter := map[string]string{
		"resource.label.container_name": container_name,
		"resource.label.pod_id":         string(pod_uid),
	}

	switch metric_type {
	case v1.ResourceMemory:
		filter["metric.type"] = "container.googleapis.com/container/memory/bytes_used"
		filter["metric.label.memory_type"] = "non-evictable"
		it := s.getTimeSeries(filter, duration)
		m = evaluateMemMetrics(it)
	case v1.ResourceCPU:
		filter["metric.type"] = "container.googleapis.com/container/cpu/usage_time"
		it := s.getTimeSeries(filter, duration)
		m = evaluateCpuMetrics(it)
	}

	m.ContainerName = container_name
	return m
}

func (s *StackDriverClient) getMetrics(pods []v1.Pod, duration time.Duration, metric_type v1.ResourceName) (metrics []*ContainerMetrics) {

	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			m := s.getContainerMetrics(container.Name, pod.ObjectMeta.UID, duration, metric_type)
			m.PodName = pod.GetName()
			metrics = append(metrics, m)
		}
	}
	return
}
