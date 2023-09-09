package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"runtime"
)

var (
	_ display.View = &System{}
)

type System struct {
}

func (s *System) Title() string {
	return "System"
}

func (s *System) Run(window fyne.Window, events <-chan display.Event) error {
	box := container.NewVBox()
	window.SetContent(box)

	// Memory Usage
	box.Add(widget.NewLabel("Memory Usage"))
	sysMemUsage := widget.NewLabel("0 B")
	allocMemUsage := widget.NewLabel("0 B")
	heapMemUsage := widget.NewLabel("0 B")
	box.Add(container.NewHBox(sysMemUsage, allocMemUsage, heapMemUsage))

	// CPU Usage
	box.Add(widget.NewLabel("CPU Usage"))
	cpuUsage := widget.NewLabel("0%")
	box.Add(cpuUsage)

	// handle events
	go func() {
		for {
			select {
			case <-events:
				// update the memory usage
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				sysMemUsage.SetText(humanReadable(uint(m.Sys)))
				allocMemUsage.SetText(humanReadable(uint(m.Alloc)))
				heapMemUsage.SetText(humanReadable(uint(m.HeapSys)))

				// update the cpu usage
				var cpuUsagePercent float64
				if m.NumGC > 0 {
					cpuUsagePercent = float64(m.PauseTotalNs) / float64(m.NumGC) / float64(10e6)
				} else {
					cpuUsagePercent = 0
				}

				cpuUsage.SetText(fmt.Sprintf("%.2f%%", cpuUsagePercent))

			}
		}
	}()

	return nil
}
