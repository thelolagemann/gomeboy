package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"image"
	"strconv"
)

type OAM struct {
	widget.BaseWidget
	PPU                            *ppu.PPU
	spriteImgs                     []*canvas.Image
	selectedSprite                 int
	selectedSpriteImage            *image.RGBA
	selectedSpriteRaster           *canvas.Raster
	selectedSpriteGrid             *widget.TextGrid
	scaleFactor, scaleFactorActive int
}

func NewOAM(p *ppu.PPU) *OAM {
	o := &OAM{PPU: p}
	o.ExtendBaseWidget(o)
	return o
}

func (o *OAM) CreateRenderer() fyne.WidgetRenderer {
	// create settings
	settings := container.NewVBox()
	// var scaleFactor = 1
	scaleDropdown := widget.NewSelect([]string{"1x", "2x", "4x", "8x"}, func(s string) {
		o.scaleFactor, _ = strconv.Atoi(s[:1])
		o.Refresh()
	})
	scaleDropdown.Selected = "4x"

	// create the grid (40 sprites, 10 sprites per row)
	grid := container.NewGridWithRows(4)

	// create the sprites
	for i := 0; i < 40; i++ {
		// create image for sprite
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		t := canvas.NewImageFromImage(img)
		t.ScaleMode = canvas.ImageScalePixels
		t.SetMinSize(fyne.NewSize(32, 32))

		o.PPU.DrawSprite(img, o.PPU.Sprites[i])

		tapImage := newWrappedTappable(func() { o.selectedSprite = i; o.Refresh() }, t)
		o.spriteImgs = append(o.spriteImgs, t)

		grid.Add(tapImage)
	}

	main := container.NewHBox()

	settings.Add(container.NewGridWithColumns(2, widget.NewLabelWithStyle("Scale: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), scaleDropdown))

	main.Add(settings)
	main.Add(container.NewVBox(grid))

	// create selected sprite
	o.selectedSpriteImage = image.NewRGBA(image.Rect(0, 0, 8, 8))
	o.selectedSpriteRaster = canvas.NewRasterFromImage(o.selectedSpriteImage)
	o.selectedSpriteRaster.ScaleMode = canvas.ImageScalePixels
	o.selectedSpriteRaster.SetMinSize(fyne.NewSize(256, 256))
	settings.Add(
		container.NewVBox(
			widget.NewLabelWithStyle("Selected Sprite", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(o.selectedSpriteRaster),
		),
	)
	o.selectedSpriteGrid = widget.NewTextGrid()
	settings.Add(container.NewVBox(widget.NewLabelWithStyle("Selected Sprite Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), o.selectedSpriteGrid))

	return widget.NewSimpleRenderer(main)
}

func (o *OAM) Refresh() {
	for i, img := range o.spriteImgs {
		o.PPU.DrawSprite(img.Image.(*image.RGBA), o.PPU.Sprites[i])
		if o.scaleFactorActive != o.scaleFactor {
			img.SetMinSize(fyne.NewSize(float32(8*o.scaleFactor), float32(8*o.scaleFactor)))
		}
		img.Refresh()
	}
	o.PPU.DrawSprite(o.selectedSpriteImage, o.PPU.Sprites[o.selectedSprite])
	o.selectedSpriteRaster.Refresh()
	o.selectedSpriteGrid.SetText(o.PPU.Sprites[o.selectedSprite].String())
	o.scaleFactorActive = o.scaleFactor
}
