package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/serial/accessories"
	"image"
	"time"
)

type printerLayout struct {
	lastY int
}

func (p printerLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	// the image should always be aligned to the top left corner
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(160*2, 200*2))

}

func (p printerLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(160*2, 200*2)
}

type Printer struct {
	*accessories.Printer
	widget.BaseWidget

	image  *image.RGBA
	raster *canvas.Image
	lastY  int
	DrawMode
}

func NewPrinter(p *accessories.Printer) *Printer {
	pr := &Printer{Printer: p}
	pr.ExtendBaseWidget(pr)
	return pr
}

func (p *Printer) CreateRenderer() fyne.WidgetRenderer {
	p.image = image.NewRGBA(image.Rect(0, 0, 160, 10))

	// create the printer image
	p.raster = canvas.NewImageFromImage(p.image)
	p.raster.SetMinSize(fyne.NewSize(160*2, 200*2))
	p.raster.ScaleMode = canvas.ImageScalePixels
	p.raster.FillMode = canvas.ImageFillStretch

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
		p.raster.Image = p.image
		p.raster.Refresh()
	})

	buttonBox.Add(saveButton)
	buttonBox.Add(clearButton)

	box := container.New(printerLayout{})

	box.Add(p.raster)

	return widget.NewSimpleRenderer(box)
}

func (p *Printer) Refresh() {
	if !p.HasPrintJob() {
		return
	}
	// process the print job
	job := p.GetPrintJob()

	// make a copy of the old image
	oldImage := p.image

	// create a new image to accommodate the new data
	p.image = image.NewRGBA(image.Rect(0, 0, 160, p.lastY+job.Bounds().Dy()))
	p.raster.Image = p.image

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
			p.raster.Refresh()
			time.Sleep(20 * time.Millisecond)
		}
	default:

	}

	// redraw the canvas
	p.raster.Refresh()

	// update the last y
	p.lastY += p.Printer.GetPrintJob().Bounds().Dy() // TODO account for white space
}

type DrawMode int

const (
	DrawModeNormal DrawMode = iota
	DrawModeLine
	DrawModePixel
)
