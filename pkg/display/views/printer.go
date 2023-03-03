package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/thelolagemann/go-gameboy/internal/serial/accessories"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
)

var (
	_ display.View = &Printer{}
)

type Printer struct {
	*accessories.Printer

	image *image.RGBA
	lastY int
}

func (p *Printer) Run(window fyne.Window, events <-chan display.Event) error {
	p.image = image.NewRGBA(image.Rect(0, 0, 160, 10))
	window.SetPadded(false)

	// create the printer image
	c := canvas.NewImageFromImage(p.image)
	c.SetMinSize(fyne.NewSize(160*4, 200*4))
	c.ScaleMode = canvas.ImageScalePixels
	c.FillMode = canvas.ImageFillContain

	// set the canvas as the content of the window
	window.SetContent(c)

	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case display.EventTypePrint:
					// process the print job
					job := e.Data.(image.Image)

					// make a copy of the old image
					oldImage := p.image

					// create a new image to accommodate the new data
					p.image = image.NewRGBA(image.Rect(0, 0, 160, p.lastY+job.Bounds().Dy()))
					c.Image = p.image

					// draw the old image onto the new one
					for x := 0; x < 160; x++ {
						for y := 0; y < p.lastY; y++ {
							p.image.Set(x, y, oldImage.At(x, y))
						}
					}

					// update the image
					for x := 0; x < 160; x++ {
						for y := 0; y < p.Printer.GetPrintJob().Bounds().Dy(); y++ {
							p.image.Set(x, p.lastY+y, job.At(x, y))
						}
					}

					// resize the canvas
					c.SetMinSize(fyne.NewSize(160*4, float32(p.lastY+p.Printer.GetPrintJob().Bounds().Dy()*4)))

					// redraw the canvas
					c.Refresh()

					// update the last y
					p.lastY += p.Printer.GetPrintJob().Bounds().Dy() // TODO account for white space
				}
			}
		}
	}()

	return nil
}
