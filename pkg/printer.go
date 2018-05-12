package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ryanuber/columnize"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"k8s.io/api/core/v1"
)

func _cmp(f1, f2 interface{}, reverse bool, field string) bool {

	if q1, ok := f1.(*CpuResource); ok {
		q2 := f2.(*CpuResource)
		v := q2.ToQuantity()
		if reverse {
			return q1.Cmp(*v) < 0
		}
		return q1.Cmp(*v) > 0
	}

	if v1, ok := f1.(int64); ok {
		v2 := f2.(int64)
		if reverse {
			return v1 < v2
		}
		return v1 > v2

	}

	if s1, ok := f1.(string); ok {
		s2 := f2.(string)
		if reverse {
			return strings.Compare(s1, s2) > 0
		}
		return strings.Compare(s1, s2) < 0
	}

	panic(fmt.Sprintf("Unknown type: _cmp %s", field))
}

func cmp(t interface{}, field string, i, j int, reverse bool) bool {

	if ra, ok := t.([]*ContainerResources); ok {
		return _cmp(getField(ra[i], field), getField(ra[j], field), reverse, field)
	}

	if cm, ok := t.([]*ContainerMetrics); ok {
		return _cmp(getField(cm[i], field), getField(cm[j], field), reverse, field)
	}

	panic("Unknown type: cmp")
}

func fmtPercent(p int64) string {
	return fmt.Sprintf("%d%%", p)
}

func FormatResourceUsage(capacity v1.ResourceList, resources []*ContainerResources, field string, reverse bool) (rows [][]string) {

	sort.Slice(resources, func(i, j int) bool {
		return cmp(resources, field, i, j, reverse)
	})

	rows = append(rows, [][]string{
		{"Namespace", "Name", "CpuReq", "CpuReq%", "CpuLimit", "CpuLimit%", "MemReq", "MemReq%", "MemLimit", "MemLimit%"},
		{"---------", "----", "------", "-------", "--------", "---------", "------", "-------", "--------", "---------"},
	}...)

	totalCpuReq, totalCpuLimit := NewCpuResource(0), NewCpuResource(0)
	totalMemoryReq, totalMemoryLimit := NewMemoryResource(0), NewMemoryResource(0)

	for _, u := range resources {
		totalCpuReq.Add(*u.CpuReq.ToQuantity())
		totalCpuLimit.Add(*u.CpuLimit.ToQuantity())
		totalMemoryReq.Add(*u.MemReq.ToQuantity())
		totalMemoryLimit.Add(*u.MemLimit.ToQuantity())

		rows = append(rows, []string{
			u.Namespace,
			u.Name,
			u.CpuReq.String(),
			fmtPercent(u.PercentCpuReq),
			u.CpuLimit.String(),
			fmtPercent(u.PercentCpuLimit),
			u.MemReq.String(),
			fmtPercent(u.PercentMemoryReq),
			u.MemLimit.String(),
			fmtPercent(u.PercentMemoryLimit),
		})
	}

	rows = append(rows, []string{"---------", "----", "------", "-------", "--------", "---------", "------", "-------", "--------", "---------"})

	cpuCapacity := NewCpuResource(capacity.Cpu().MilliValue())
	memoryCapacity := NewMemoryResource(capacity.Memory().Value())

	rows = append(rows, []string{
		"Total",
		"",
		fmt.Sprintf("%s/%s", totalCpuReq.String(), cpuCapacity.String()),
		fmtPercent(totalCpuReq.calcPercentage(capacity.Cpu())),
		fmt.Sprintf("%s/%s", totalCpuLimit.String(), cpuCapacity.String()),
		fmtPercent(totalCpuLimit.calcPercentage(capacity.Cpu())),
		fmt.Sprintf("%s/%s", totalMemoryReq.String(), memoryCapacity.String()),
		fmtPercent(totalMemoryReq.calcPercentage(capacity.Memory())),
		fmt.Sprintf("%s/%s", totalMemoryLimit.String(), memoryCapacity.String()),
		fmtPercent(totalMemoryLimit.calcPercentage(capacity.Memory())),
	})

	return rows
}

func ExportCSVResourceUsage(rows [][]string) {
	w := csv.NewWriter(os.Stdout)
	w.WriteAll(rows)

	if err := w.Error(); err != nil {
		log.Fatalln("error writing csv:", err)
	}

}

func PrintResourceUsage(rows [][]string) {
	var formatted []string

	for _, row := range rows {
		formatted = append(formatted, strings.Join(row, " | "))
	}

	fmt.Println(columnize.SimpleFormat(formatted))
}

func PrintContainerMetrics(containerMetrics []*ContainerMetrics, metric_type v1.ResourceName, duration time.Duration, field string, reverse bool) {

	sort.Slice(containerMetrics, func(i, j int) bool {
		return cmp(containerMetrics, field, i, j, reverse)
	})

	var mode_or_avg string

	switch metric_type {
	case v1.ResourceMemory:
		mode_or_avg = "Mode"
	case v1.ResourceCPU:
		mode_or_avg = "Avg"
	}

	table := []string{
		fmt.Sprintf("                        Pod/Container                         |  Last  |   Min  |   Max  | %s", mode_or_avg),
		"------------------------------------------------------------- | ------ | ------ | ------ | --------",
	}

	var total int64
	for _, m := range containerMetrics {
		row := []string{
			fmt.Sprintf("%s/%s", m.PodName, m.ContainerName),
		}
		s := m.toSlice()
		row = append(row, s...)
		table = append(table, strings.Join(row, " | "))
		total += m.DataPoints
	}

	p := message.NewPrinter(language.English)

	fmt.Println(columnize.SimpleFormat(table))
	fmt.Printf("\nResults shown are for a period of %s. %s data points were evaluted.\n", duration.String(), p.Sprintf("%d", total))
}
