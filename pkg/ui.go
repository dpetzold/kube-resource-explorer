package main

import (
	"fmt"

	ui "github.com/airking05/termui"
)

func EventWidget() (events *ui.Par) {
	events = ui.NewPar("<> This row has 3 columns\n<- Widgets can be stacked up like left side\n<- Stacked widgets are treated as a single widget")
	events.Height = 20
	events.BorderLabel = "Events"

	return events
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

func TopInit(nodes []string) {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	node_gauges := make(map[string]map[string]*ui.Gauge)

	for _, node := range nodes {
		node_gauges[node] = make(map[string]*ui.Gauge)
		node_gauges[node]["cpu"] = GaugeWidget("Cpu", ui.ColorRed)
		node_gauges[node]["mem"] = GaugeWidget("Mem", ui.ColorCyan)
	}

	var cpu_column []ui.GridBufferer
	var mem_column []ui.GridBufferer

	for _, gauge := range node_gauges {
		cpu_column = append(cpu_column, gauge["cpu"])
		mem_column = append(mem_column, gauge["mem"])
	}

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(3, 0, ListWidget(nodes)),
			ui.NewCol(3, 0, cpu_column...),
			ui.NewCol(3, 0, mem_column...),
		),
		ui.NewRow(
			ui.NewCol(9, 0, EventWidget()),
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

		/*
			for _, g := range gs {
				g.Percent = (g.Percent + 3) % 100
			}
		*/

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
