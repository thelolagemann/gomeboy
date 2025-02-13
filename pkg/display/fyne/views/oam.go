package views

import (
	"bytes"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"image"
	"strconv"
)

type OAM struct {
	widget.BaseWidget
	PPU         *ppu.PPU
	b           *io.Bus
	spriteImgs  []*canvas.Image
	spriteTiles []Tile

	selectedSprite                 int
	selectedSpriteImage            *image.RGBA
	selectedSpriteRaster           *canvas.Raster
	selectedSpriteGrid             *widget.TextGrid
	scaleFactor, scaleFactorActive int
}

func NewOAM(p *ppu.PPU, b *io.Bus) *OAM {
	o := &OAM{PPU: p, spriteTiles: make([]Tile, 40), b: b}
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
		o.spriteTiles[i] = Tile{}

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
			widget.NewLabelWithStyle("Selected Object", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(o.selectedSpriteRaster),
		),
	)
	o.selectedSpriteGrid = widget.NewTextGrid()
	settings.Add(container.NewVBox(widget.NewLabelWithStyle("Selected Object Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), o.selectedSpriteGrid))

	return widget.NewSimpleRenderer(main)
}

func (o *OAM) Refresh() {
	for i, img := range o.spriteImgs {
		// get the tile id from bus
		tileID := o.b.Get(0xfe00 + uint16(i)<<2 + 2)
		data := getTileData(o.b, 0, int(tileID), 0)
		if bytes.Equal(data, o.spriteTiles[i]) {
			continue
		}
		o.spriteTiles[i] = data

		o.spriteTiles[i].Draw(o.spriteImgs[i].Image.(*image.RGBA), 0, 0, o.PPU.ColourOBJPalette[0])
		if o.scaleFactorActive != o.scaleFactor {
			img.SetMinSize(fyne.NewSize(float32(8*o.scaleFactor), float32(8*o.scaleFactor)))
		}
		img.Refresh()
	}
	o.spriteTiles[o.selectedSprite].Draw(o.selectedSpriteImage, 0, 0, o.PPU.ColourOBJPalette[0])
	o.selectedSpriteRaster.Refresh()
	address := 0xfe00 + uint16(o.selectedSprite<<2)
	o.selectedSpriteGrid.SetText(fmt.Sprintf("Y: %d\nX: %d\nID: %02x\nAttributes: %08b\n", o.b.Get(address), o.b.Get(address+1), o.b.Get(address+2), o.b.Get(address+3)))
	o.scaleFactorActive = o.scaleFactor
}

// A Tile has a size of 8x8 pixels, using a 2bpp format.
type Tile []uint8

// Draw draws the tile to the given image at the given position.
func (t Tile) Draw(img *image.RGBA, i int, i2 int, pal ppu.Palette) {
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			copy(img.Pix[((i2+y)*img.Stride)+((i+x)*4):], append(pal[(t[y]>>(7-x)&1)|(t[y+8]>>(7-x)&1)<<1][:], 0xff))
		}
	}
}
