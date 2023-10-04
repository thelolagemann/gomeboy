package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/thelolagemann/gomeboy/internal/apu"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
	"image"
)

type Visualizer struct {
	apu *apu.APU
}

func (v *Visualizer) Title() string {
	return "Visualizer"
}

func (v *Visualizer) Run(window fyne.Window, events <-chan event.Event) error {
	// create a grid
	grid := container.NewVBox()
	window.SetContent(grid)

	// create a plot for channel 1
	channel1Plot := plot.New()
	channel1Plot.Title.Text = "Channel 1 (Square)"
	channel1Plot.X.Label.Text = "Time"
	channel1Plot.Y.Label.Text = "Amplitude"

	channel1Image := image.NewRGBA(image.Rect(0, 0, 640, 480))
	c := vgimg.NewWith(vgimg.UseImage(channel1Image))
	channel1Plot.Draw(draw.New(c))

	channel1Canvas := canvas.NewRasterFromImage(c.Image())
	channel1Canvas.ScaleMode = canvas.ImageScalePixels
	channel1Canvas.SetMinSize(fyne.NewSize(640, 480))

	grid.Add(channel1Canvas)

	// create a line for the plot
	line, err := plotter.NewLine(make(plotter.XYs, 16))
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case event.Quit:
					return
				case event.Sample:
					// get the samples from event
					samples := e.Data.(apu.Samples)
					// update the plot

					lastVal := uint8(0) // last value of the sample (so the plot doesn't diagonally go up)
					lastX := 0.0        // last X value of the plot (so the plot doesn't diagonally go up)
					for i := 0; i < 16; i++ {
						// if lastVal not equal, then keep the same X value , but change the Y value
						if lastVal != samples[i].Channel1 {
							line.XYs[i].X = lastX
							line.XYs[i].Y = float64(samples[i].Channel1)
						} else {
							lastX = float64(i)
							line.XYs[i].X = float64(i)
							line.XYs[i].Y = float64(samples[i].Channel1)
						}
						lastVal = samples[i].Channel1
					}

					// redraw the plot
					channel1Plot.Add(line)
					channel1Plot.Draw(draw.New(c))
					channel1Canvas.Refresh()
				}
			}
		}
	}()

	return nil
}
