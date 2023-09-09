package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"image"
)

type Tilemaps struct {
	PPU *ppu.PPU
}

func (t *Tilemaps) Title() string {
	return "Tilemaps"
}

func (t *Tilemaps) Run(window fyne.Window, events <-chan display.Event) error {
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

	var segmentTiles = 0
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
	tilemap0Image := image.NewRGBA(image.Rect(0, 0, 256, 256))
	tilemap0ImageCanvas := canvas.NewRasterFromImage(tilemap0Image)
	tilemap0ImageCanvas.ScaleMode = canvas.ImageScalePixels
	tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))
	tilemap0Tap := newTappableImage(tilemap0Image, tilemap0ImageCanvas, func(e *fyne.PointEvent) {
		// using the position, we need to calculate which tile was clicked
		// 1. get the position of the click
		// 2. divide the position by the scale factor
		// 3. divide the position by 8 (tile size)
		// 4. get the tile at that position
		// 5. draw a box around the tile
		// 6. draw the tile in the tile viewer
		realX := e.Position.X / float32(scaleFactor)
		realY := e.Position.Y / float32(scaleFactor)

		tileX := int(realX / 8)
		tileY := int(realY / 8)

		tileIndex := tileX + (tileY * 32)

		// get the tile entry from the tilemap
		tileEntry := t.PPU.TileMaps[0][tileX][tileY]

		// get the tile from the tile entry

		fmt.Printf("BG Priority: %v\nXFlip: %t\nYFlip: %t\nTile Number: %d\nBank: %d", tileEntry.Attributes.BGPriority, tileEntry.Attributes.XFlip, tileEntry.Attributes.YFlip, tileIndex, tileEntry.Attributes.VRAMBank)
	})
	tilemap0Content.Add(tilemap0Tap)

	tilemap0.Add(tilemap0Content)

	// tilemap 1
	tilemap1 := container.NewVBox(widget.NewLabelWithStyle("Tilemap 1 (0x9C00 - 0x9FFF)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	tilemap1Content := container.NewHBox()

	// tilemap image
	tilemap1Image := image.NewRGBA(image.Rect(0, 0, 256, 256))
	tilemap1ImageCanvas := canvas.NewRasterFromImage(tilemap1Image)
	tilemap1ImageCanvas.ScaleMode = canvas.ImageScalePixels
	tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))
	tilemap1Content.Add(tilemap1ImageCanvas)

	tilemap1.Add(tilemap1Content)

	// add tilemaps to map container
	mapGrid.Add(tilemap0)
	mapGrid.Add(tilemap1)

	window.SetContent(main)

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
		tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))
		tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))

		tilemap0ImageCanvas.Refresh()
		tilemap1ImageCanvas.Refresh()
	}

	segmentTilesCheckbox.OnChanged = func(b bool) {

		// if segmentTiles is true, we need to segment the tiles
		// by first recreating the tilemap images to accomodate
		// the gaps in between the tiles
		if segmentTiles == 0 && b {
			// remove the tilemap images from the map container
			tilemap0Content.RemoveAll()
			tilemap1Content.RemoveAll()

			tilemap0Image = image.NewRGBA(image.Rect(0, 0, 384, 384))
			tilemap0ImageCanvas = canvas.NewRasterFromImage(tilemap0Image)
			tilemap0ImageCanvas.ScaleMode = canvas.ImageScalePixels
			tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(384*scaleFactor), float32(384*scaleFactor)))

			tilemap1Image = image.NewRGBA(image.Rect(0, 0, 384, 384))
			tilemap1ImageCanvas = canvas.NewRasterFromImage(tilemap1Image)
			tilemap1ImageCanvas.ScaleMode = canvas.ImageScalePixels
			tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(384*scaleFactor), float32(384*scaleFactor)))

			tilemap0Content.Add(tilemap0ImageCanvas)
			tilemap1Content.Add(tilemap1ImageCanvas)

			segmentTiles = 4
		} else if segmentTiles != 0 && !b {
			// remove the tilemap images from the map container
			tilemap0Content.RemoveAll()
			tilemap1Content.RemoveAll()

			tilemap0Image = image.NewRGBA(image.Rect(0, 0, 256, 256))
			tilemap0ImageCanvas = canvas.NewRasterFromImage(tilemap0Image)
			tilemap0ImageCanvas.ScaleMode = canvas.ImageScalePixels
			tilemap0ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))

			tilemap1Image = image.NewRGBA(image.Rect(0, 0, 256, 256))
			tilemap1ImageCanvas = canvas.NewRasterFromImage(tilemap1Image)
			tilemap1ImageCanvas.ScaleMode = canvas.ImageScalePixels
			tilemap1ImageCanvas.SetMinSize(fyne.NewSize(float32(256*scaleFactor), float32(256*scaleFactor)))

			tilemap0Content.Add(tilemap0ImageCanvas)
			tilemap1Content.Add(tilemap1ImageCanvas)

			segmentTiles = 0
		}

	}

	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case display.EventTypeQuit:
					return
				case display.EventTypeFrame:
					// update tilemap 0
					t.PPU.DumpTileMaps(tilemap0Image, tilemap1Image, segmentTiles)
					tilemap0ImageCanvas.Refresh()
					tilemap1ImageCanvas.Refresh()
				}
			}
		}
	}()

	return nil
}
