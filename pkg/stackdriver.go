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
