package views

import (
	"bytes"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display/fyne/themes"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"image"
	"image/color"
	"strconv"
	"sync"
)

type Tiles struct {
	sync.Mutex
	widget.BaseWidget
	*ppu.PPU
	bus *io.Bus

	tileImages      [768]*image.RGBA
	tileWidgets     [768]*tappable
	selectedPalette *ppu.Palette

	tiles       [2][384]Tile
	lastPalette ppu.Palette
}

func NewTiles(p *ppu.PPU) *Tiles {
	t := &Tiles{PPU: p}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Tiles) CreateRenderer() fyne.WidgetRenderer {
	// create main container
	main := container.NewHBox()

	// create settings container
	settings := container.NewVBox()

	var scaleFactor = 2
	scaleDropdown := widget.NewSelect([]string{"1x", "2x", "3x", "4x"}, nil)
	scaleDropdown.Selected = "2x"
	settings.Add(container.NewGridWithColumns(2, widget.NewLabelWithStyle("Scale  ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), scaleDropdown))

	// create paletteView selection container
	t.selectedPalette = &t.PPU.ColourPalette[0]
	paletteSelection := container.NewGridWithColumns(2)
	paletteSelection.Add(widget.NewLabelWithStyle("Palette", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	var paletteOptions [16]string
	for i := 0; i < 8; i++ {
		paletteOptions[i] = fmt.Sprintf("BG %d", i)
		paletteOptions[i+8] = fmt.Sprintf("OBJ %d", i)
	}

	// create paletteView selection dropdown
	paletteSelectionDropdown := widget.NewSelect(paletteOptions[:], nil)
	paletteSelectionDropdown.Selected = "BG 0"
	paletteSelection.Add(paletteSelectionDropdown)
	settings.Add(paletteSelection)

	// create selected tile container
	selectedTile := container.NewVBox()
	selectedTile.Add(widget.NewLabelWithStyle("Selected Tile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	// create selected tile grid (8x8 tiles, 1 tappable rectangle per pixel of the tile)
	selectedTileGrid := container.NewVBox()

	var showNumbers = false
	var selectedTileIndex = 0
	var selectedTileBank = 0
	recreateSelectedTile := func(show bool) {
		selectedTileGrid.RemoveAll()
		if show {
			buttonGrid := container.NewVBox()
			for i := 0; i < 8; i++ {
				row := container.NewHBox()
				for j := 0; j < 8; j++ {
					high, low := t.tiles[selectedTileBank][selectedTileIndex][j], t.tiles[selectedTileBank][selectedTileIndex][j+8]
					text := canvas.NewText(strconv.Itoa(int((high>>(7-i))&1)|int((low>>(7-i))&1)<<1), themeColor(theme.ColorNameForeground))
					text.TextSize = 15
					text.TextStyle.Monospace = true
					text.Alignment = fyne.TextAlignCenter
					row.Add(newBadge(themeColor(themes.ColorNameBackgroundOnBackground), 5, container.NewPadded(text)))
				}
				buttonGrid.Add(row)
			}

			selectedTileGrid.Add(buttonGrid)
		} else {
			recGrid := container.NewVBox()
			// add 8x8 tappable rectangles to the selected tile grid
			for i := 0; i < 8; i++ {
				row := container.NewHBox()
				for j := 0; j < 8; j++ {
					r := canvas.NewRectangle(color.White)
					r.SetMinSize(fyne.NewSize(24, 24))
					row.Add(newWrappedTappable(nil, r))
					high, low := t.tiles[selectedTileBank][selectedTileIndex][j], t.tiles[selectedTileBank][selectedTileIndex][j+8]
					rgb := t.selectedPalette[int((high>>(7-i))&1)|int((low>>(7-i))&1)<<1]

					r.FillColor = color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}
				}
				recGrid.Add(row)
			}
			selectedTileGrid.Add(recGrid)
		}
		showNumbers = show
	}
	recreateSelectedTile(false)

	// add checkbox for viewing colour numbers rather than colours
	viewColourNumbers := widget.NewCheck("Show Colour Numbers", func(b bool) { recreateSelectedTile(b) })
	settings.Add(viewColourNumbers)

	// add the selected tile grid to the selected tile container
	selectedTile.Add(selectedTileGrid)

	// create selected tile info container
	selectedTileInfo := container.NewVBox()
	selectedTileInfo.Add(widget.NewLabelWithStyle("Tile Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	tileInfoTextGrid := widget.NewTextGrid()
	tileInfoTextGrid.SetText(`Index	0
Address	0x8000`)
	selectedTileInfo.Add(tileInfoTextGrid)
	selectedTile.Add(selectedTileInfo)

	// create action box for the settings container
	actionBox := container.NewGridWithRows(4)

	// create selected actions container
	selectedActions := container.NewGridWithColumns(2)
	actionBox.Add(selectedActions)

	// add copy/export buttons to the selected actions container
	selectedActions.Add(widget.NewButton("Copy", func() {
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		t.tiles[selectedTileBank][selectedTileIndex].Draw(img, 0, 0, *t.selectedPalette)
		showError(utils.CopyImage(img), "Tile Viewer")
	}))
	selectedActions.Add(widget.NewButton("Save", func() {
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		t.tiles[selectedTileBank][selectedTileIndex].Draw(img, 0, 0, *t.selectedPalette)
		saveImage(img, fmt.Sprintf("%d-%04x.png", selectedTileBank, selectedTileIndex), "Tile Viewer")
	}))

	// create all actions container
	allActions := container.NewGridWithColumns(2)
	actionBox.Add(allActions)

	// add copy/export buttons to the all actions container
	allActions.Add(widget.NewButton("Copy All", func() { showError(utils.CopyImage(t.getTiles(256, 192, true, true)), "Tile Viewer") }))
	allActions.Add(widget.NewButton("Save All", func() { saveImage(t.getTiles(256, 192, true, true), "all_tiles.png", "Tile Viewer") }))

	// create bank 0 container
	bank0Actions := container.NewGridWithColumns(2)
	actionBox.Add(bank0Actions)

	// add copy/export buttons to the bank 0 container
	bank0Actions.Add(widget.NewButton("Copy Bank 0", func() { showError(utils.CopyImage(t.getTiles(128, 192, true, false)), "Tile Viewer") }))
	bank0Actions.Add(widget.NewButton("Save Bank 0", func() { saveImage(t.getTiles(128, 192, true, false), "bank0_tiles.png", "Tile Viewer") }))

	// create bank 1 container
	bank1Actions := container.NewGridWithColumns(2)
	actionBox.Add(bank1Actions)

	// add copy/export buttons to the bank 1 container
	bank1Actions.Add(widget.NewButton("Copy Bank 1", func() { showError(utils.CopyImage(t.getTiles(128, 192, false, true)), "Tile Viewer") }))
	bank1Actions.Add(widget.NewButton("Save Bank 1", func() { saveImage(t.getTiles(128, 192, false, true), "bank1_tiles.png", "Tile Viewer") }))

	// add the action box to the settings container
	selectedTile.Add(actionBox)
	settings.Add(selectedTile)

	main.Add(settings)

	bankGrid := container.NewHBox()
	main.Add(bankGrid)

	// bank 0
	bank0 := container.NewVBox(widget.NewLabelWithStyle("Bank 0", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	bank0Content := container.NewHBox()

	// tile box
	tileGrid0 := container.NewGridWithRows(3)
	grid1 := container.NewGridWithColumns(16) // 0x8000 - 0x8800
	grid2 := container.NewGridWithColumns(16) // 0x8800 - 0x9000
	grid3 := container.NewGridWithColumns(16) // 0x9000 - 0x9800

	// add the grids to the tile grid
	tileGrid0.Add(grid1)
	tileGrid0.Add(grid2)
	tileGrid0.Add(grid3)
	bank0Content.Add(tileGrid0)
	bank0.Add(bank0Content)

	// add the tile grid to the bank grid
	bankGrid.Add(bank0)

	// bank 1
	bank1 := container.NewVBox(widget.NewLabelWithStyle("Bank 1", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	bank1Content := container.NewHBox()

	// tile box
	tileGrid1 := container.NewGridWithRows(3)
	grid4 := container.NewGridWithColumns(16) // 0x8000 - 0x8800
	grid5 := container.NewGridWithColumns(16) // 0x8800 - 0x9000
	grid6 := container.NewGridWithColumns(16) // 0x9000 - 0x9800

	// add the grids to the tile grid
	tileGrid1.Add(grid4)
	tileGrid1.Add(grid5)
	tileGrid1.Add(grid6)
	bank1Content.Add(tileGrid1)
	bank1.Add(bank1Content)

	// add the tile grid to the bank grid
	bankGrid.Add(bank1)

	var selectTile = func(bank, tile int) {
		// set the selected tile and bank
		selectedTileIndex = tile
		selectedTileBank = bank
		recreateSelectedTile(showNumbers)
		selectedTileGrid.Refresh()

		// update the tile info textgrid
		tileInfoTextGrid.SetText(`Index	` + strconv.Itoa(tile) + `
Address	` + strconv.Itoa(bank) + `:0x` + fmt.Sprintf("%X", 0x8000+(tile*16)))

		// refresh the selected tile
		selectedTile.Refresh()
	}

	var recreateTiles = func() {
		t.Lock()
		defer t.Unlock()

		// remove all tiles from the grids
		grid1.RemoveAll()
		grid2.RemoveAll()
		grid3.RemoveAll()
		grid4.RemoveAll()
		grid5.RemoveAll()
		grid6.RemoveAll()

		// create the tiles
		for i := 0; i < 384; i++ {
			bank0Img := image.NewRGBA(image.Rect(0, 0, 8, 8))
			bank1Img := image.NewRGBA(image.Rect(0, 0, 8, 8))
			bank0Raster, bank1Raster := canvas.NewRasterFromImage(bank0Img), canvas.NewRasterFromImage(bank1Img)
			bank0Raster.ScaleMode, bank1Raster.ScaleMode = canvas.ImageScalePixels, canvas.ImageScalePixels
			bank0Raster.SetMinSize(fyne.NewSize(float32(8*scaleFactor), float32(8*scaleFactor)))
			bank1Raster.SetMinSize(fyne.NewSize(float32(8*scaleFactor), float32(8*scaleFactor)))

			t.tiles[0][i].Draw(bank0Img, 0, 0, *t.selectedPalette)
			t.tiles[1][i].Draw(bank1Img, 0, 0, *t.selectedPalette)

			// add the tile to the grid
			bank0Tap := newWrappedTappable(func() { selectTile(0, i) }, bank0Raster)
			bank1Tap := newWrappedTappable(func() { selectTile(1, i) }, bank1Raster)
			switch {
			case i < 128:
				grid1.Add(bank0Tap)
				grid4.Add(bank1Tap)
			case i < 256:
				grid2.Add(bank0Tap)
				grid5.Add(bank1Tap)
			default:
				grid3.Add(bank0Tap)
				grid6.Add(bank1Tap)
			}

			// add the tile to the tile images
			t.tileImages[i], t.tileImages[i+384] = bank0Img, bank1Img
			t.tileWidgets[i], t.tileWidgets[i+384] = bank0Tap, bank1Tap
		}
	}
	recreateTiles()

	// event handlers
	paletteSelectionDropdown.OnChanged = func(s string) {
		paletteNumber, _ := strconv.Atoi(s[len(s)-1:])
		if s[0:2] == "BG" {
			t.selectedPalette = &t.PPU.ColourPalette[paletteNumber]
		} else {
			t.selectedPalette = &t.PPU.ColourSpritePalette[paletteNumber]
		}
		selectTile(selectedTileBank, selectedTileIndex)
		t.Refresh()
	}
	scaleDropdown.OnChanged = func(value string) {
		scaleFactor, _ = strconv.Atoi(value[:1])
		recreateTiles()
	}

	return widget.NewSimpleRenderer(main)
}

func (t *Tiles) Refresh() {
	t.Lock()
	defer t.Unlock()
	var paletteChanged = *t.selectedPalette != t.lastPalette
	for i, img := range t.tileImages {
		if i < 384 {
			if paletteChanged || !bytes.Equal(getTileData(t.bus, i, 0)[:], t.tiles[0][i][:]) {
				t.tiles[0][i].Draw(img, 0, 0, *t.selectedPalette)
			}
		} else {
			if paletteChanged || !bytes.Equal(getTileData(t.bus, i, 1), t.tiles[1][i-384][:]) {
				t.tiles[1][i-384].Draw(img, 0, 0, *t.selectedPalette)
			}
		}
		t.tileWidgets[i].Refresh()
	}

	t.lastPalette = *t.selectedPalette
}

func getTileData(b *io.Bus, bank int, index int) Tile {
	var t, tT Tile = make(Tile, 16), make(Tile, 16)
	address := uint16(0x0000) | uint16(index)<<4
	for i := uint16(0); i < 16; i++ {
		t[i] = b.GetVRAM(address+i, uint8(bank))
	}

	for i := 0; i < 16; i++ {
		if i%2 == 0 {
			tT[i/2] = t[i]
		} else {
			tT[8+i/2] = t[i]
		}
	}

	return tT
}

func (t *Tiles) getTiles(w, h int, bank0, bank1 bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for i := 0; i < 384; i++ {
		switch {
		case bank0 && bank1:
			t.tiles[0][i].Draw(img, (i%16)*8, i/16*8, *t.selectedPalette)
			t.tiles[1][i].Draw(img, 128+(i%16)*8, i/16*8, *t.selectedPalette)
		case bank0:
			t.tiles[0][i].Draw(img, (i%16)*8, i/16*8, *t.selectedPalette)
		case bank1:
			t.tiles[1][i].Draw(img, (i%16)*8, i/16*8, *t.selectedPalette)
		}
	}

	return img
}
