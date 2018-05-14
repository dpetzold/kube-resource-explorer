package main

import (
	"fmt"

	ui "github.com/airking05/termui"
	"github.com/davecgh/go-spew/spew"
	api_v1 "k8s.io/api/core/v1"
)

func EventWidget(events string) *ui.Par {
	event := ui.NewPar(events)
	event.Height = 20
	event.BorderLabel = "Events"

	return event
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
	gauge.BorderLabel = label
	gauge.Height = 3
	gauge.Percent = 0
	gauge.PaddingBottom = 0
	gauge.BarColor = barColor
	gauge.BorderFg = ui.ColorWhite
	gauge.BorderLabelFg = ui.ColorCyan

	return gauge
}

func TopInit(k *KubeClient, nodes []*api_v1.Node) {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	node_gauges := make(map[*api_v1.Node]map[string]*ui.Gauge)
	var node_names []string

	for _, node := range nodes {
		node_gauges[node] = make(map[string]*ui.Gauge)
		node_gauges[node]["cpu"] = GaugeWidget("Cpu", ui.ColorRed)
		node_gauges[node]["mem"] = GaugeWidget("Mem", ui.ColorCyan)
		node_names = append(node_names, node.GetName())
	}

	events := spew.Sdump(len(node_gauges))

	var cpu_column []ui.GridBufferer
	var mem_column []ui.GridBufferer

	for _, gauge := range node_gauges {
		cpu_column = append(cpu_column, gauge["cpu"])
		mem_column = append(mem_column, gauge["mem"])
	}

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(3, 0, ListWidget(node_names)),
			ui.NewCol(3, 0, cpu_column...),
			ui.NewCol(3, 0, mem_column...),
		),
		ui.NewRow(
			ui.NewCol(9, 0, EventWidget(events)),
		),
	)

	ui.Body.Align()

	ui.Render(ui.Body)

	ui.Handle("/sys/kbd/q", func(ui.Event) {
		ui.StopLoop()
	})

	ui.Handle("/timer/1s", func(e ui.Event) {
		// t := e.Data.(ui.EvtTimer)
		// i := t.Count

		for node, g := range node_gauges {
			r, _ := k.NodeResources(node)
			g["mem"].Percent = r.PercentMemory
			g["cpu"].Percent = r.PercentCpu
		}

		// sp.Lines[0].Data = spdata[:100+i]
		// lc.Data = sinps[2*i:]

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
