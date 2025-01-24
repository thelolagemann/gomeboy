package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"image"
)

type FIFO struct {
	widget.BaseWidget

	p   *ppu.PPU
	img *image.RGBA
	c   *canvas.Image
}

func NewFIFO(p *ppu.PPU) *FIFO {
	f := &FIFO{p: p}
	f.ExtendBaseWidget(f)
	return f
}

func (f *FIFO) CreateRenderer() fyne.WidgetRenderer {
	// create image
	f.img = image.NewRGBA(image.Rect(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))
	f.c = canvas.NewImageFromImage(f.img)
	f.c.ScaleMode = canvas.ImageScalePixels
	f.c.SetMinSize(fyne.NewSize(ppu.ScreenWidth*4, ppu.ScreenHeight*4))

	return widget.NewSimpleRenderer(container.NewVBox(f.c))
}

func (f *FIFO) Refresh() {
	for y := 0; y < ppu.ScreenHeight; y++ {
		for x := 0; x < ppu.ScreenWidth; x++ {
			copy(f.img.Pix[y*ppu.ScreenWidth+x:], f.p.DebugView[y][x][:])
			f.img.Pix[y*ppu.ScreenWidth+x+3] = 0xff
		}
	}

	f.c.Refresh()
}
