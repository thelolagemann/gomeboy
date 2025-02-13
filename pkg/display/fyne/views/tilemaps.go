package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/internal/types"
	"image"
)

type Tilemaps struct {
	widget.BaseWidget
	PPU *ppu.PPU
	b   *io.Bus

	tilemapData [2][32][32]Tile

	tilemap0ImageCanvas, tilemap1ImageCanvas *canvas.Raster
	tilemap0Image, tilemap1Image             *image.RGBA
	segmentTiles                             int
}

func NewTilemaps(p *ppu.PPU, b *io.Bus) *Tilemaps {
	t := &Tilemaps{PPU: p, b: b}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Tilemaps) CreateRenderer() fyne.WidgetRenderer {
	// create main container
	main := container.NewHBox()

	// create settings container
	settings := container.NewVBox()

	var scaleFactor = 2
	// create scale dropdown
	scaleDropdown := widget.NewSelect([]string{"1x", "2x", "3x", "4x"}, nil)
	scaleDropdown.SetSelected("2x")

	// add the scale dropdown to the settings container
	settings.Add(container.NewGridWithColumns(2,
		widget.NewLabelWithStyle("Scale	", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		scaleDropdown,
	))

	t.segmentTiles = 0
	// create segment tiles checkbox
	segmentTilesCheckbox := widget.NewCheck("", nil)

	// add the segment tiles checkbox to the settings container
	settings.Add(container.NewGridWithColumns(2,
		widget.NewLabelWithStyle("Segment Tiles", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		segmentTilesCheckbox,
	))

	main.Add(settings)

	// create map container
	mapGrid := container.NewHBox()
	main.Add(mapGrid)

	// tilemap 0
	tilemap0 := container.NewVBox(widget.NewLabelWithStyle("Tilemap 0 (0x9800 - 0x9BFF)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	tilemap0Content := container.NewHBox()

	// tilemap image
	t.tilemap0Image = image.NewRGBA(image.Rect(0, 0, 256, 256))
	t.tilemap0ImageCanvas = canvas.NewRasterFromImage(t.tilemap0Image)
	t.tilemap0ImageCanvas.ScaleMode = canvas.ImageScalePixels
	t.tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))
	tilemap0Tap := newWrappedTappable(func() {}, t.tilemap0ImageCanvas)
	tilemap0Content.Add(tilemap0Tap)

	tilemap0.Add(tilemap0Content)

	// tilemap 1
	tilemap1 := container.NewVBox(widget.NewLabelWithStyle("Tilemap 1 (0x9C00 - 0x9FFF)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	tilemap1Content := container.NewHBox()

	// tilemap image
	t.tilemap1Image = image.NewRGBA(image.Rect(0, 0, 256, 256))
	t.tilemap1ImageCanvas = canvas.NewRasterFromImage(t.tilemap1Image)
	t.tilemap1ImageCanvas.ScaleMode = canvas.ImageScalePixels
	t.tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))
	tilemap1Content.Add(t.tilemap1ImageCanvas)

	tilemap1.Add(tilemap1Content)

	// add tilemaps to map container
	mapGrid.Add(tilemap0)
	mapGrid.Add(tilemap1)

	// event handlers
	scaleDropdown.OnChanged = func(s string) {
		// resize tilemap images based on scale
		switch s {
		case "1x":
			scaleFactor = 1
		case "2x":
			scaleFactor = 2
		case "3x":
			scaleFactor = 3
		case "4x":
			scaleFactor = 4
		}
		t.tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))
		t.tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))

		t.tilemap0ImageCanvas.Refresh()
		t.tilemap1ImageCanvas.Refresh()
	}

	segmentTilesCheckbox.OnChanged = func(b bool) {

		// if segmentTiles is true, we need to segment the tiles
		// by first recreating the tilemap images to accomodate
		// the gaps in between the tiles
		if t.segmentTiles == 0 && b {
			// remove the tilemap images from the map container
			tilemap0Content.RemoveAll()
			tilemap1Content.RemoveAll()

			t.tilemap0Image = image.NewRGBA(image.Rect(0, 0, 384, 384))
			t.tilemap0ImageCanvas = canvas.NewRasterFromImage(t.tilemap0Image)
			t.tilemap0ImageCanvas.ScaleMode = canvas.ImageScalePixels
			t.tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(384*scaleFactor), float32(384*scaleFactor)))

			t.tilemap1Image = image.NewRGBA(image.Rect(0, 0, 384, 384))
			t.tilemap1ImageCanvas = canvas.NewRasterFromImage(t.tilemap1Image)
			t.tilemap1ImageCanvas.ScaleMode = canvas.ImageScalePixels
			t.tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(384*scaleFactor), float32(384*scaleFactor)))

			tilemap0Content.Add(t.tilemap0ImageCanvas)
			tilemap1Content.Add(t.tilemap1ImageCanvas)

			t.segmentTiles = 4
		} else if t.segmentTiles != 0 && !b {
			// remove the tilemap images from the map container
			tilemap0Content.RemoveAll()
			tilemap1Content.RemoveAll()

			t.tilemap0Image = image.NewRGBA(image.Rect(0, 0, 256, 256))
			t.tilemap0ImageCanvas = canvas.NewRasterFromImage(t.tilemap0Image)
			t.tilemap0ImageCanvas.ScaleMode = canvas.ImageScalePixels
			t.tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))

			t.tilemap1Image = image.NewRGBA(image.Rect(0, 0, 256, 256))
			t.tilemap1ImageCanvas = canvas.NewRasterFromImage(t.tilemap1Image)
			t.tilemap1ImageCanvas.ScaleMode = canvas.ImageScalePixels
			t.tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))

			tilemap0Content.Add(t.tilemap0ImageCanvas)
			tilemap1Content.Add(t.tilemap1ImageCanvas)

			t.segmentTiles = 0
		}

	}

	return widget.NewSimpleRenderer(main)
}

func (t *Tilemaps) Refresh() {
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			// get tile id from tilemap data
			tileID := int(t.b.VRAM[0][0x1800+(y*32)+x])

			// 32 x 32 tile maps
			newTileData := getTileData(t.b, 0, int(tileID), 1&^(t.b.Get(types.LCDC)&types.Bit4>>4)&1)

			t.tilemapData[0][y][x] = newTileData
			t.tilemapData[0][y][x].Draw(t.tilemap0Image, x*8, y*8, t.PPU.ColourPalette[0])

			tileID = int(t.b.VRAM[0][0x1C00+(y*32)+x])
			newTileData = getTileData(t.b, 0, int(tileID), 1&^(t.b.Get(types.LCDC)&types.Bit4>>4)&1)

			t.tilemapData[1][y][x] = newTileData
			t.tilemapData[1][y][x].Draw(t.tilemap1Image, x*8, y*8, t.PPU.ColourPalette[0])
		}
	}

	// t.PPU.DumpTileMaps(t.tilemap0Image, t.tilemap1Image, t.segmentTiles)
	t.tilemap0ImageCanvas.Refresh()
	t.tilemap1ImageCanvas.Refresh()
}
