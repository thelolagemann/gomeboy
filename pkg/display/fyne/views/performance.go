package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"image"
	"time"
)

type Performance struct {
}

func (p *Performance) Title() string {
	return "Performance"
}

func (p *Performance) Run(window fyne.Window, events <-chan event.Event) error {
	// create the base view
	grid := container.NewVBox()
	window.SetContent(grid)

	// create plot for the frametime
	frameTimePlot := plot.New()
	frameTimePlot.Title.Text = "Frame Time"

	plotters := make(plotter.XYs, 100)
	line, err := plotter.NewLine(plotters)
	if err != nil {
		panic(err)
	}

	// create an image for the frametime
	frameTimeImage := image.NewRGBA(image.Rect(0, 0, 640, 480))

	c := vgimg.NewWith(vgimg.UseImage(frameTimeImage))
	frameTimePlot.Draw(draw.New(c))

	frameTimeCanvas := canvas.NewRasterFromImage(c.Image())
	frameTimeCanvas.ScaleMode = canvas.ImageScalePixels
	frameTimeCanvas.SetMinSize(fyne.NewSize(640, 480))

	// add the image to the grid
	grid.Add(frameTimeCanvas)

	go func() {

		for {
			select {
			case e := <-events:
				switch e.Type {
				case event.Quit:
					return
				case event.FrameTime:
					// get the list of frame times from the event
					frameTimes := e.Data.([]time.Duration)

					for i, frameTime := range frameTimes {
						line.XYs[i].X = float64(i)
						line.XYs[i].Y = float64(frameTime)
						if frameTime == 0 {
							fmt.Println("frame time is 0")
						}
					}

					// redraw the plot
					frameTimePlot.Add(line)
					frameTimePlot.Draw(draw.New(c))
					frameTimeCanvas.Refresh()

				}

			}
		}
	}()

	return nil
}
