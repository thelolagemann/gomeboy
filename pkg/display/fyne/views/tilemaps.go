package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"image"
)

type Tilemaps struct {
	widget.BaseWidget
	PPU *ppu.PPU

	tilemap0ImageCanvas, tilemap1ImageCanvas *canvas.Raster
	tilemap0Image, tilemap1Image             *image.RGBA
	segmentTiles                             int
}

func NewTilemaps(p *ppu.PPU) *Tilemaps {
	t := &Tilemaps{PPU: p}
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
	tilemap0Tap := newTappableImage(t.tilemap0Image, t.tilemap0ImageCanvas, func(e *fyne.PointEvent) {
		// using the position, we need to calculate which tile was clicked
		// 1. get the position of the click
		// 2. divide the position by the scale factor
		// 3. divide the position by 8 (tile size)
		// 4. get the tile at that position
		// 5. draw a box around the tile
		// 6. draw the tile in the tile viewer
		//realX := e.Position.X / float32(scaleFactor)
		//realY := e.Position.Y / float32(scaleFactor)

		//tileX := int(realX / 8)
		//tileY := int(realY / 8)

		//tileIndex := tileX + (tileY * 32)

		// get the tile entry from the tilemap
		//tileEntry := t.PPU.TileMaps[0][tileX][tileY]

		// get the tile from the tile entry

		//fmt.Printf("BG Priority: %v\nXFlip: %t\nYFlip: %t\nTile Number: %d\nBank: %d", tileEntry.Attributes.BGPriority, tileEntry.Attributes.XFlip, tileEntry.Attributes.YFlip, tileIndex, tileEntry.Attributes.VRAMBank)
	})
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
	t.PPU.DumpTileMaps(t.tilemap0Image, t.tilemap1Image, t.segmentTiles)
	t.tilemap0ImageCanvas.Refresh()
	t.tilemap1ImageCanvas.Refresh()
}
