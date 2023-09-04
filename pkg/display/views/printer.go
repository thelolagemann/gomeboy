package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/serial/accessories"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
	"time"
)

var (
	_ display.View = &Printer{}
)

type printerLayout struct {
	lastY int
}

func (p printerLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	fmt.Println(len(objects), size)
	// the image should always be aligned to the top left corner
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(160*2, 200*2))

}

func (p printerLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(160*2, 200*2)
}

type Printer struct {
	*accessories.Printer

	image *image.RGBA
	lastY int
	DrawMode
}

func (p *Printer) Title() string {
	return "Printer"
}

type DrawMode int

const (
	DrawModeNormal DrawMode = iota
	DrawModeLine
	DrawModePixel
)

func (p *Printer) Run(window fyne.Window, events <-chan display.Event) error {
	p.image = image.NewRGBA(image.Rect(0, 0, 160, 10))
	window.SetPadded(false)

	// create the printer image
	c := canvas.NewImageFromImage(p.image)
	c.SetMinSize(fyne.NewSize(160*2, 200*2))
	c.ScaleMode = canvas.ImageScalePixels
	c.FillMode = canvas.ImageFillStretch

	// create a box for the buttons
	buttonBox := container.NewHBox()

	// create a button to save the image
	saveButton := widget.NewButton("Save", func() {
		p.PrintStashed() // TODO ask user where to save
	})

	// create a button to clear the image
	clearButton := widget.NewButton("Clear", func() {
		p.image = image.NewRGBA(image.Rect(0, 0, 160, 10))
		p.lastY = 0
		c.Image = p.image
		c.Refresh()
	})

	buttonBox.Add(saveButton)
	buttonBox.Add(clearButton)

	box := container.New(printerLayout{})

	box.Add(c)

	// set the canvas as the content of the window
	window.SetContent(box)

	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case display.EventTypeQuit:
					return
				case display.EventTypePrint:
					// process the print job
					job := e.Data.(image.Image)

					// make a copy of the old image
					oldImage := p.image

					// create a new image to accommodate the new data
					p.image = image.NewRGBA(image.Rect(0, 0, 160, p.lastY+job.Bounds().Dy()))
					c.Image = p.image

					switch p.DrawMode {
					case DrawModeNormal:
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
					case DrawModeLine:
						// draw the old image onto the new one
						for x := 0; x < 160; x++ {
							for y := 0; y < p.lastY; y++ {
								p.image.Set(x, y, oldImage.At(x, y))
							}
						}

						for y := 0; y < p.Printer.GetPrintJob().Bounds().Dy(); y++ {
							for x := 0; x < 160; x++ {
								p.image.Set(x, p.lastY+y, p.Printer.GetPrintJob().At(x, y))
							}
							c.Refresh()
							time.Sleep(20 * time.Millisecond)
						}
					}

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
