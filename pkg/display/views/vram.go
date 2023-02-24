package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image"
	"strconv"
	"strings"
)

var (
	_ display.View = &VRAM{}
)

type VRAM struct {
	*ppu.PPU
}

func (v *VRAM) Run(window display.Window) error {
	bankGrid := container.NewHBox()

	// bank 0
	bank0 := container.NewVBox(widget.NewLabelWithStyle("Bank 0", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	bank0Content := container.NewHBox()

	// tile box
	tileGrid0 := container.NewGridWithRows(3)

	labelGrid := container.NewVBox()

	// 1st set (0x8000 - 0x8800)
	grid1 := container.NewGridWithColumns(16)

	// 2nd set (0x8800 - 0x9000)
	grid2 := container.NewGridWithColumns(16)

	// 3rd set (0x9000 - 0x9800)
	grid3 := container.NewGridWithColumns(16)

	// add the grids to the tile grid
	tileGrid0.Add(grid1)
	tileGrid0.Add(grid2)
	tileGrid0.Add(grid3)

	bank0Content.Add(labelGrid)
	bank0Content.Add(tileGrid0)

	bank0.Add(bank0Content)

	// add the tile grid to the bank grid
	bankGrid.Add(bank0)

	// bank 1
	bank1 := container.NewVBox(widget.NewLabelWithStyle("Bank 1", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	bank1Content := container.NewHBox()

	// tile box
	tileGrid1 := container.NewGridWithRows(3)

	labelGrid1 := container.NewVBox()

	// 1st set (0x8000 - 0x8800)
	grid4 := container.NewGridWithColumns(16)

	// 2nd set (0x8800 - 0x9000)
	grid5 := container.NewGridWithColumns(16)

	// 3rd set (0x9000 - 0x9800)
	grid6 := container.NewGridWithColumns(16)

	// add the grids to the tile grid
	tileGrid1.Add(grid4)
	tileGrid1.Add(grid5)
	tileGrid1.Add(grid6)

	bank1Content.Add(labelGrid1)
	bank1Content.Add(tileGrid1)

	bank1.Add(bank1Content)
	// add the tile grid to the bank grid
	bankGrid.Add(bank1)

	var tileImages []*image.RGBA

	// create the tiles
	for i := 0; i < 384; i++ {
		// should we add a label?
		if i%16 == 0 {
			// add labels to grid (e.g. 0x8000, 0x8100, 0x8200, ...) next to the tiles
			labelGrid.Add(widget.NewLabelWithStyle("0x"+strings.ToUpper(strconv.FormatInt(int64(0x8000+i*16), 16)), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		}

		// create the tile image
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		t := canvas.NewImageFromImage(img)
		t.ScaleMode = canvas.ImageScalePixels
		t.SetMinSize(fyne.NewSize(32, 32))

		// add the tile to the grid
		if i < 128 {
			grid1.Add(t)
		} else if i < 256 {
			grid2.Add(t)
		} else {
			grid3.Add(t)
		}

		tileImages = append(tileImages, img)
	}

	// create the tiles (bank 1)
	for i := 0; i < 384; i++ {
		// should we add a label?
		if i%16 == 0 {
			// add labels to grid (e.g. 0x8000, 0x8100, 0x8200, ...) next to the tiles
			labelGrid1.Add(widget.NewLabelWithStyle("0x"+strings.ToUpper(strconv.FormatInt(int64(0x8000+i*16), 16)), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		}
		// create the tile image
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		t := canvas.NewImageFromImage(img)
		t.ScaleMode = canvas.ImageScalePixels
		t.SetMinSize(fyne.NewSize(32, 32))

		// add the tile to the grid
		if i < 128 {
			grid4.Add(t)
		} else if i < 256 {
			grid5.Add(t)
		} else {
			grid6.Add(t)
		}

		tileImages = append(tileImages, img)
	}

	// tile map
	tileMap := container.NewVBox(widget.NewLabelWithStyle("Tile Maps", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	tileMapImages := make([]*image.RGBA, 2)

	// create the tile map images
	for i := 0; i < 2; i++ {
		// create the tile map image
		img := image.NewRGBA(image.Rect(0, 0, 256, 256))
		t := canvas.NewRasterFromImage(img)
		t.ScaleMode = canvas.ImageScalePixels
		t.SetMinSize(fyne.NewSize(512, 512))

		tileMapImages[i] = img
		tileMap.Add(t)
	}

	// add the tile map to the bank grid
	bankGrid.Add(tileMap)

	// set the content of the window
	window.FyneWindow().SetContent(bankGrid)

	// handle events
	go func() {
		for {
			select {
			case <-window.Events():
				// update the tiles
				for i, img := range tileImages {
					// update the tile
					if i >= 384 {
						v.PPU.TileData[1][i-384].Draw(img, 0, 0)
					} else {
						v.PPU.TileData[0][i].Draw(img, 0, 0)
					}
				}

				// refresh the tile grid
				tileGrid0.Refresh()
				tileGrid1.Refresh()

				// update the tile maps
				v.PPU.DumpTileMaps(tileMapImages[0], tileMapImages[1])

				// refresh the tile map
				tileMap.Refresh()
			}
		}
	}()

	return nil
}
