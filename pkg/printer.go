package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ryanuber/columnize"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"k8s.io/apimachinery/pkg/api/resource"
)

func _cmp(f1, f2 interface{}, reverse bool, field string) bool {

	if q1, ok := f1.(*resource.Quantity); ok {
		q2 := f2.(*resource.Quantity)
		if reverse {
			return q1.Cmp(*q2) < 0
		}
		return q1.Cmp(*q2) > 0
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

	if ra, ok := t.([]*ResourceAllocation); ok {
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

func PrintResourceUsage(resourceUsage []*ResourceAllocation, field string, reverse bool) {

	sort.Slice(resourceUsage, func(i, j int) bool {
		return cmp(resourceUsage, field, i, j, reverse)
	})

	rows := []string{
		"Namespace | Name | CpuReq | CpuReq% | CpuLimit | CpuLimit% | MemReq | MemReq% | MemLimit | MemLimit%",
		"--------- | ---- | ------ | ------- | -------- | --------- | ------ | ------- | -------- | ---------",
	}

	for _, u := range resourceUsage {
		row := strings.Join([]string{
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
		}, "| ")
		rows = append(rows, row)
	}

	fmt.Println(columnize.SimpleFormat(rows))
}

func PrintContainerMetrics(containerMetrics []*ContainerMetrics, duration time.Duration, field string, reverse bool) {

	sort.Slice(containerMetrics, func(i, j int) bool {
		return cmp(containerMetrics, field, i, j, reverse)
	})

	table := []string{
		"                        Pod/Container                         |  Last  |   Min  |   Max  | Avg/Mode",
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
	fmt.Printf("\n\nResults shown are for a period of %s. %s data points were analyzed.\n\n", duration.String(), p.Sprintf("%d", total))
}
