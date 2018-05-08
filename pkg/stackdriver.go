package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	log "github.com/Sirupsen/logrus"

	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/types"

	"github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"k8s.io/apimachinery/pkg/api/resource"
)

type StackDriverClient struct {
	ctx       context.Context
	client    *monitoring.MetricClient
	project   string
	clientset *kubernetes.Clientset
}

func NewStackDriverClient(project string, clientset *kubernetes.Clientset) *StackDriverClient {
	ctx := context.Background()
	c, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		panic(err.Error())
	}
	return &StackDriverClient{
		ctx:       ctx,
		client:    c,
		project:   project,
		clientset: clientset,
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

func (c *StackDriverClient) getTimeSeries(filter_map map[string]string, duration time.Duration) *monitoring.TimeSeriesIterator {

	filter := buildTimeSeriesFilter(filter_map)

	log.Debug(filter)

	end := time.Now().UTC()
	start := end.Add(-duration)

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   fmt.Sprintf("projects/%s", c.project),
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

	return c.client.ListTimeSeries(c.ctx, req)
}

func (c *StackDriverClient) getMemMetrics(it *monitoring.TimeSeriesIterator) *ContainerMetrics {

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

func (c *StackDriverClient) getCpuMetrics(it *monitoring.TimeSeriesIterator) *ContainerMetrics {
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

func (c *StackDriverClient) getContainerMetrics(container_name string, pod_uid types.UID, duration time.Duration, metric_type MetricType) *ContainerMetrics {

	var m *ContainerMetrics

	switch metric_type {
	case MEM:
		filter := map[string]string{
			"metric.type":                   "container.googleapis.com/container/memory/bytes_used",
			"resource.label.container_name": container_name,
			"resource.label.pod_id":         string(pod_uid),
			"metric.label.memory_type":      "non-evictable",
		}
		it := c.getTimeSeries(filter, duration)
		m = c.getMemMetrics(it)
	case CPU:
		filter := map[string]string{
			"metric.type":                   "container.googleapis.com/container/cpu/usage_time",
			"resource.label.container_name": container_name,
			"resource.label.pod_id":         string(pod_uid),
		}

		it := c.getTimeSeries(filter, duration)
		m = c.getCpuMetrics(it)
	}

	m.ContainerName = container_name
	return m
}
