// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

// +build ignore

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
	list.Height = 8

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

func main() {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	gs := make([]*ui.Gauge, 6)

	for i := 0; i < 3; i++ {
		gs[i] = GaugeWidget("Cpu", ui.ColorRed)
	}

	for i := 3; i < 6; i++ {
		gs[i] = GaugeWidget("Mem", ui.ColorCyan)
	}

	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(3, 0, ListWidget([]string{"node1", "node2", "node3"})),
			ui.NewCol(3, 0, gs[0], gs[1], gs[2]),
			ui.NewCol(3, 0, gs[3], gs[4], gs[5]),
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

		for _, g := range gs {
			g.Percent = (g.Percent + 3) % 100
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
