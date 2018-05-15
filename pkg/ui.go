package main

import (
	"fmt"
	"sort"

	ui "github.com/airking05/termui"
	"github.com/davecgh/go-spew/spew"
	api_v1 "k8s.io/api/core/v1"
)

type NodeDisplay struct {
	Node        api_v1.Node
	CpuGauge    *ui.Gauge
	MemoryGauge *ui.Gauge
}

func EventsWidget(events string) *ui.Par {
	w := ui.NewPar(events)
	w.Height = 20
	w.BorderLabel = "Events"
	return w
}

func PodsWidget() *ui.Table {
	w := ui.NewTable()
	w.Height = 30
	w.BorderLabel = "Pods"
	w.TextAlign = ui.AlignLeft
	w.Separator = false
	w.Analysis()
	w.SetSize()
	return w
}

func ListWidget(labels []string) *ui.List {

	var items []string
	for i, label := range labels {
		items = append(items, []string{
			"",
			fmt.Sprintf("[%d] %s", i+1, label),
			"",
		}...)
	}

	w := ui.NewList()
	w.Border = false
	w.Items = items
	w.Height = len(labels) * 3
	return w
}

func GaugeWidget(label string, barColor ui.Attribute) *ui.Gauge {

	w := ui.NewGauge()
	w.BarColor = barColor
	w.BorderFg = ui.ColorWhite
	w.BorderLabelFg = ui.ColorCyan
	w.BorderLabel = label
	w.Height = 3
	w.LabelAlign = ui.AlignRight
	w.PaddingBottom = 0
	w.Percent = 0
	return w
}

func TopInit(k *KubeClient) {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	nodes, err := k.Nodes()
	if err != nil {
		panic(err.Error())
	}

	node_gauges := make(map[string]*NodeDisplay)
	var node_names []string

	for _, node := range nodes {
		name := node.GetName()
		node_gauges[name] = &NodeDisplay{
			Node:        node,
			CpuGauge:    GaugeWidget("Cpu", ui.ColorRed),
			MemoryGauge: GaugeWidget("Mem", ui.ColorCyan),
		}
		node_names = append(node_names, name[len(name)-26:])
	}

	events := spew.Sdump(node_names)

	var cpu_column []ui.GridBufferer
	var mem_column []ui.GridBufferer

	for _, nd := range node_gauges {
		cpu_column = append(cpu_column, nd.CpuGauge)
		mem_column = append(mem_column, nd.MemoryGauge)
	}

	listWidget := ListWidget(node_names)
	podsWidget := PodsWidget()
	eventsWidget := EventsWidget(events)

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(3, 0, listWidget),
			ui.NewCol(3, 0, cpu_column...),
			ui.NewCol(3, 0, mem_column...),
		),
		ui.NewRow(
			ui.NewCol(9, 0, podsWidget),
		),
		ui.NewRow(
			ui.NewCol(9, 0, eventsWidget),
		),
	)

	ui.Body.Align()
	ui.Render(ui.Body)

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(e ui.Event) {

		for _, nd := range node_gauges {
			r, _ := k.NodeResourceUsage(&nd.Node)
			nd.MemoryGauge.Percent = r.PercentMemory
			nd.MemoryGauge.Label = fmt.Sprintf("%d%% (%s)", r.PercentMemory, r.MemoryUsage.String())
			nd.CpuGauge.Percent = r.PercentCpu
			nd.CpuGauge.Label = fmt.Sprintf("%d%% (%s)", r.PercentCpu, r.CpuUsage.String())
		}

		metrics, err := k.PodResourceUsage("")
		if err != nil {
			panic(err.Error())
		}

		sort.Slice(metrics, func(i, j int) bool {
			q := metrics[j].CpuUsage.ToQuantity()
			return metrics[i].CpuUsage.ToQuantity().Cmp(*q) > 0
		})

		var nmax int
		for _, m := range metrics {
			if len(m.Name) > nmax {
				nmax = len(m.Name)
			}
		}

		pods := [][]string{
			[]string{"Pod/Container", "Cpu", "Memory"},
			[]string{"", "", ""},
		}

		for _, m := range metrics {
			pods = append(pods, []string{m.Name, m.CpuUsage.String(), m.MemoryUsage.String()})
		}

		podsWidget.Rows = pods
		podsWidget.Analysis()
		podsWidget.SetSize()

		ui.Body.Align()

		ui.Render(ui.Body)
	})

	ui.Handle("/sys/wnd/resize", func(e ui.Event) {
		ui.Body.Width = ui.TermWidth()
		ui.Body.Align()
		ui.Clear()
		ui.Render(ui.Body)
	})

	ui.Loop()
}
