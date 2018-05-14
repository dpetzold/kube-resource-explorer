package main

import (
	"fmt"

	ui "github.com/airking05/termui"
	"github.com/davecgh/go-spew/spew"
	api_v1 "k8s.io/api/core/v1"
)

func EventsWidget(events string) *ui.Par {
	widget := ui.NewPar(events)
	widget.Height = 20
	widget.BorderLabel = "Events"
	return widget
}

func PodsWidget(events string) *ui.Par {
	widget := ui.NewPar(events)
	widget.Height = 20
	widget.BorderLabel = "Pods"
	return widget
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

	list := ui.NewList()
	list.Border = false
	list.Items = items
	list.Height = len(labels) * 3

	return list
}

func GaugeWidget(label string, barColor ui.Attribute) *ui.Gauge {

	gauge := ui.NewGauge()
	gauge.BarColor = barColor
	gauge.BorderFg = ui.ColorWhite
	gauge.BorderLabelFg = ui.ColorCyan
	gauge.BorderLabel = label
	gauge.Height = 3
	gauge.LabelAlign = ui.AlignRight
	gauge.PaddingBottom = 0
	gauge.Percent = 0
	return gauge
}

type NodeDisplay struct {
	Node        api_v1.Node
	CpuGauge    *ui.Gauge
	MemoryGauge *ui.Gauge
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
	podsWidget := PodsWidget("")
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
			r, _ := k.NodeResources(&nd.Node)
			nd.MemoryGauge.Percent = r.PercentMemory
			nd.MemoryGauge.Label = fmt.Sprintf("%d%% (%s)", r.PercentMemory, r.MemoryUsage.String())
			nd.CpuGauge.Percent = r.PercentCpu
			nd.CpuGauge.Label = fmt.Sprintf("%d%% (%s)", r.PercentMemory, r.CpuUsage.String())
		}

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
