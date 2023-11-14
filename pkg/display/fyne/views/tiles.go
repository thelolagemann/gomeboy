package views

import (
	"bytes"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"image"
	"image/color"
	"strconv"
	"strings"
)

type Tiles struct {
	*ppu.PPU

	tiles [2][384]ppu.Tile
}

func (v *Tiles) Title() string {
	return "Tiles"
}

func (v *Tiles) Run(window fyne.Window, events <-chan event.Event) error {
	// create main container
	main := container.NewHBox()

	// create settings container
	settings := container.NewVBox()

	var scaleFactor = 2
	// create scale dropdown
	scaleDropdown := widget.NewSelect([]string{"1x", "2x", "3x", "4x"}, nil)
	scaleDropdown.Selected = "2x"

	// add the magnification slider to the settings container
	settings.Add(container.NewGridWithColumns(2, widget.NewLabelWithStyle("Scale  ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), scaleDropdown))

	// create paletteView selection container
	var selectedPalette = v.PPU.ColourPalette.Palettes[0]
	paletteSelection := container.NewGridWithColumns(2)
	paletteSelection.Add(widget.NewLabelWithStyle("Palette", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	paletteOptions := []string{}

	for i := 0; i < 8; i++ {
		paletteOptions = append(paletteOptions, fmt.Sprintf("BG %d", i))
	}

	for i := 0; i < 8; i++ {
		paletteOptions = append(paletteOptions, fmt.Sprintf("OBJ %d", i))
	}

	// create paletteView selection dropdown
	paletteSelectionDropdown := widget.NewSelect(paletteOptions, nil)

	// add the paletteView selection dropdown to the paletteView selection container
	paletteSelection.Add(paletteSelectionDropdown)

	// add the paletteView selection container to the settings container
	settings.Add(paletteSelection)

	// create selected tile container
	selectedTile := container.NewVBox()
	selectedTile.Add(widget.NewLabelWithStyle("Selected Tile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	// create selected tile grid (8x8 tiles, 1 tappable rectangle per pixel of the tile)
	selectedTileGrid := container.NewVBox()

	var showNumbers = false
	var selectedTileIndex = 0
	var selectedTileBank = 0
	recreateSelectedTile := func() {
		selectedTileGrid.RemoveAll()

		if showNumbers {
			buttonGrid := container.NewGridWithColumns(8)

			for i := 0; i < 8; i++ {
				row := container.NewGridWithRows(8)
				for j := 0; j < 8; j++ {
					textNumber := newCustomPaddedButton("", nil)

					row.Add(textNumber)

					var x = i
					var y = j
					high, low := v.PPU.TileData[selectedTileBank][selectedTileIndex][y], v.PPU.TileData[selectedTileBank][selectedTileIndex][y+8]
					var colourNum = int((high >> (7 - x)) & 1)
					colourNum |= int((low>>(7-x))&1) << 1

					textNumber.Button.Text = strconv.Itoa(colourNum)
				}
				buttonGrid.Add(row)
			}

			selectedTileGrid.Add(buttonGrid)
		} else {
			recGrid := container.NewGridWithColumns(8)
			// add 8x8 tappable rectangles to the selected tile grid
			for i := 0; i < 8; i++ {
				row := container.NewGridWithRows(8)
				for j := 0; j < 8; j++ {
					rect := newTappableRectangle(nil)
					rect.rec.SetMinSize(fyne.NewSize(26, 26))
					row.Add(rect)
					var x = i
					var y = j
					high, low := v.PPU.TileData[selectedTileBank][selectedTileIndex][y], v.PPU.TileData[selectedTileBank][selectedTileIndex][y+8]
					var colourNum = int((high >> (7 - x)) & 1)
					colourNum |= int((low>>(7-x))&1) << 1
					rgb := selectedPalette.GetColour(uint8(colourNum))

					rect.rec.FillColor = color.RGBA{
						R: rgb[0],
						G: rgb[1],
						B: rgb[2],
						A: 255,
					}
				}
				recGrid.Add(row)
			}
			selectedTileGrid.Add(recGrid)
		}
	}

	recreateSelectedTile()

	// add checkbox for viewing colour numbers rather than colours
	viewColourNumbers := widget.NewCheck("Show Colour Numbers", func(b bool) {
		showNumbers = b

		recreateSelectedTile()
	})

	settings.Add(viewColourNumbers)

	// add the selected tile grid to the selected tile container
	selectedTile.Add(selectedTileGrid)

	// create selected tile info container
	selectedTileInfo := container.NewVBox()
	selectedTileInfo.Add(widget.NewLabelWithStyle("Tile Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))

	// create tile info textgrid
	tileInfoTextGrid := widget.NewTextGrid()
	tileInfoTextGrid.SetText(`Index	0
Address	0x8000`)

	// add the tile info textgrid to the selected tile info container
	selectedTileInfo.Add(tileInfoTextGrid)

	// add the tile info container to the settings container
	selectedTile.Add(selectedTileInfo)

	// create action box for the settings container
	actionBox := container.NewGridWithRows(4)

	// create selected actions container
	selectedActions := container.NewGridWithColumns(2)
	actionBox.Add(selectedActions)

	// add copy/export buttons to the selected actions container
	selectedActions.Add(widget.NewButton("Copy", func() {
		/*if err := utils.CopyImage(selectedTileImg); err != nil {
			panic(err)
		}*/
	}))
	selectedActions.Add(widget.NewButton("Save", func() {
		/*if err := utils.SaveImage(selectedTileImg); err != nil {
			panic(err)
		}*/
	}))

	// create all actions container
	allActions := container.NewGridWithColumns(2)
	actionBox.Add(allActions)

	// add copy/export buttons to the all actions container
	allActions.Add(widget.NewButton("Copy All", func() {
		// create a new image to draw the tiles on
		img := image.NewRGBA(image.Rect(0, 0, 256, 192)) // 24x32 tiles

		// draw the tiles onto the image
		for i := 0; i < 384; i++ {
			v.PPU.TileData[0][i].Draw(img, (i%16)*8, i/16*8, selectedPalette)
			v.PPU.TileData[1][i].Draw(img, 128+(i%16)*8, i/16*8, selectedPalette)
		}

		// copy the image to the clipboard
		if err := utils.CopyImage(img); err != nil {
			panic(err)
		}
	}))
	allActions.Add(widget.NewButton("Save All", func() {
		// create a new image to draw the tiles on
		img := image.NewRGBA(image.Rect(0, 0, 256, 192)) // 32x24 tiles

		// draw the tiles onto the image
		for i := 0; i < 384; i++ {
			v.PPU.TileData[0][i].Draw(img, (i%16)*8, i/16*8, selectedPalette)
			v.PPU.TileData[1][i].Draw(img, 128+(i%16)*8, i/16*8, selectedPalette)
		}

		// save the image to the file
		if err := utils.SaveImage(img); err != nil {
			panic(err)
		}
	}))

	// create bank 0 container
	bank0Actions := container.NewGridWithColumns(2)
	actionBox.Add(bank0Actions)

	// add copy/export buttons to the bank 0 container
	bank0Actions.Add(widget.NewButton("Copy Bank 0", func() {
		// create a new image to draw the tiles on
		img := image.NewRGBA(image.Rect(0, 0, 128, 192)) // 16x24 tiles

		// draw the tiles onto the image
		for i := 0; i < 384; i++ {
			v.PPU.TileData[0][i].Draw(img, (i%16)*8, i/16*8, selectedPalette)
		}

		// copy the image to the clipboard
		if err := utils.CopyImage(img); err != nil {
			panic(err)
		}
	}))
	bank0Actions.Add(widget.NewButton("Save Bank 0", func() {
		// create a new image to draw the tiles on
		img := image.NewRGBA(image.Rect(0, 0, 128, 192)) // 16x24 tiles

		// draw the tiles onto the image
		for i := 0; i < 384; i++ {
			v.PPU.TileData[0][i].Draw(img, (i%16)*8, i/16*8, selectedPalette)
		}

		// save the image to the file
		if err := utils.SaveImage(img); err != nil {
			panic(err)
		}
	}))

	// create bank 1 container
	bank1Actions := container.NewGridWithColumns(2)
	actionBox.Add(bank1Actions)

	// add copy/export buttons to the bank 1 container
	bank1Actions.Add(widget.NewButton("Copy Bank 1", func() {
		// create a new image to draw the tiles on
		img := image.NewRGBA(image.Rect(0, 0, 128, 192)) // 16x24 tiles

		// draw the tiles onto the image
		for i := 0; i < 384; i++ {
			v.PPU.TileData[1][i].Draw(img, (i%16)*8, i/16*8, selectedPalette)
		}

		// copy the image to the clipboard
		if err := utils.CopyImage(img); err != nil {
			panic(err)
		}
	}))
	bank1Actions.Add(widget.NewButton("Save Bank 1", func() {
		// create a new image to draw the tiles on
		img := image.NewRGBA(image.Rect(0, 0, 128, 192)) // 16x24 tiles

		// draw the tiles onto the image
		for i := 0; i < 384; i++ {
			v.PPU.TileData[1][i].Draw(img, (i%16)*8, i/16*8, selectedPalette)
		}

		// save the image to the file
		if err := utils.SaveImage(img); err != nil {
			panic(err)
		}
	}))

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

	bank0Content.Add(tileGrid0)

	bank0.Add(bank0Content)

	// add the tile grid to the bank grid
	bankGrid.Add(bank0)

	// bank 1
	bank1 := container.NewVBox(widget.NewLabelWithStyle("Bank 1", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	bank1Content := container.NewHBox()

	// tile box
	tileGrid1 := container.NewGridWithRows(3)

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

	bank1Content.Add(tileGrid1)

	bank1.Add(bank1Content)
	// add the tile grid to the bank grid
	bankGrid.Add(bank1)

	var tileImages []*tappableImage

	var selectTile = func(bank, tile int) {
		// set the selected tile and bank
		selectedTileIndex = tile
		selectedTileBank = bank

		// draw the tile by setting the rectangle colors to the tile colors
		if showNumbers {
			butGrid := selectedTileGrid.Objects[0].(*fyne.Container)
			for i := 0; i < 8; i++ {
				for j := 0; j < 8; j++ {

					var x = i
					var y = j
					high, low := v.PPU.TileData[bank][tile][y], v.PPU.TileData[bank][tile][y+8]
					var colourNum = int((high >> (7 - x)) & 1)
					colourNum |= int((low>>(7-x))&1) << 1

					butGrid.Objects[x].(*fyne.Container).Objects[y].(*customPaddedButton).Button.Text = fmt.Sprintf("%d", colourNum)
				}
			}
		} else {
			recGrid := selectedTileGrid.Objects[0].(*fyne.Container)
			for i := 0; i < 8; i++ {
				for j := 0; j < 8; j++ {

					var x = i
					var y = j
					high, low := v.PPU.TileData[bank][tile][y], v.PPU.TileData[bank][tile][y+8]
					var colourNum = int((high >> (7 - x)) & 1)
					colourNum |= int((low>>(7-x))&1) << 1
					rgb := selectedPalette.GetColour(uint8(colourNum))

					recGrid.Objects[x].(*fyne.Container).Objects[y].(*tappableRectangle).rec.FillColor = color.RGBA{
						R: rgb[0],
						G: rgb[1],
						B: rgb[2],
						A: 255,
					}
				}
			}
		}
		selectedTileGrid.Refresh()

		// update the tile info textgrid
		tileInfoTextGrid.SetText(`Index	` + strconv.Itoa(tile) + `
Address	` + strconv.Itoa(bank) + `:0x` + fmt.Sprintf("%X", 0x8000+(tile*16)))

		// refresh the selected tile
		selectedTile.Refresh()
	}

	var recreateTiles = func() {
		// empty the tile images
		tileImages = []*tappableImage{}

		// remove all tiles from the grids
		grid1.RemoveAll()
		grid2.RemoveAll()
		grid3.RemoveAll()
		grid4.RemoveAll()
		grid5.RemoveAll()
		grid6.RemoveAll()

		// create the tiles
		for i := 0; i < 384; i++ {
			newI := i

			// create the tile image
			img := image.NewRGBA(image.Rect(0, 0, 8, 8))
			t := canvas.NewRasterFromImage(img)
			t.ScaleMode = canvas.ImageScalePixels
			t.SetMinSize(fyne.NewSize(float32(8*scaleFactor), float32(8*scaleFactor)))

			// draw the tile
			v.PPU.TileData[0][i].Draw(img, 0, 0, selectedPalette)

			var tapImage *tappableImage
			// add the tile to the grid
			if i < 128 {
				tapImage = newTappableImage(img, t, func(_ *fyne.PointEvent) {
					// update the selected tile
					selectTile(0, newI)
				})
				grid1.Add(tapImage)
			} else if i < 256 {
				tapImage = newTappableImage(img, t, func(_ *fyne.PointEvent) {
					// update the selected tile
					selectTile(0, newI)
				})
				grid2.Add(tapImage)
			} else {
				tapImage = newTappableImage(img, t, func(_ *fyne.PointEvent) {
					// update the selected tile
					selectTile(0, newI)
				})
				grid3.Add(tapImage)
			}

			// add the tile to the tile images
			tileImages = append(tileImages, tapImage)
		}

		// create the tiles (bank 1)
		for i := 0; i < 384; i++ {
			newI := i
			// create the tile image
			img := image.NewRGBA(image.Rect(0, 0, 8, 8))
			t := canvas.NewRasterFromImage(img)
			t.ScaleMode = canvas.ImageScalePixels
			t.SetMinSize(fyne.NewSize(float32(8*scaleFactor), float32(8*scaleFactor)))

			// draw the tile
			v.PPU.TileData[1][i].Draw(img, 0, 0, selectedPalette)

			var tapImage *tappableImage
			// add the tile to the grid
			if i < 128 {
				tapImage = newTappableImage(img, t, func(_ *fyne.PointEvent) {
					// update the selected tile
					selectTile(1, newI)
				})
				grid4.Add(tapImage)

			} else if i < 256 {
				tapImage = newTappableImage(img, t, func(_ *fyne.PointEvent) {
					// update the selected tile
					selectTile(1, newI)
				})
				grid5.Add(tapImage)
			} else {
				tapImage = newTappableImage(img, t, func(_ *fyne.PointEvent) {
					// update the selected tile
					selectTile(1, newI)
				})
				grid6.Add(tapImage)
			}

			// add the tile to the tile images
			tileImages = append(tileImages, tapImage)
		}
	}

	recreateTiles()
	// set the content of the window
	window.SetContent(main)

	// event handlers
	paletteSelectionDropdown.OnChanged = func(s string) {
		// is it BG or OBJ?
		if s[0:2] == "BG" {
			// get number
			paletteNumber, _ := strconv.Atoi(s[3:])
			selectedPalette = v.PPU.ColourPalette.Palettes[paletteNumber]
		} else {
			paletteNumber, _ := strconv.Atoi(s[4:])
			selectedPalette = v.PPU.ColourPalette.Palettes[paletteNumber]
		}

		// get text grid contents
		textGridContents := strings.Split(tileInfoTextGrid.Text(), "\n")

		// get selected tile number TODO this is very cursed and can be improved
		selectedTileBank, _ := strconv.Atoi(textGridContents[1][8:9])
		selectedTileNumber, _ := strconv.Atoi(textGridContents[0][6:])

		selectTile(selectedTileBank, selectedTileNumber)

		// refresh the selected tile
		selectedTile.Refresh()

		// refresh the tile grids
		recreateTiles()
	}
	scaleDropdown.OnChanged = func(value string) {
		switch value {
		case "1x":
			scaleFactor = 1
		case "2x":
			scaleFactor = 2
		case "3x":
			scaleFactor = 3
		case "4x":
			scaleFactor = 4
		}
		recreateTiles()

		// resize window to fit the new tiles
		window.Resize(main.MinSize())
	}

	// handle event
	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case event.Quit:
					return
				case event.FrameTime:
					for i, img := range tileImages {
						if i < 384 {
							if !bytes.Equal(v.PPU.TileData[0][i][:], v.tiles[0][i][:]) {
								// update the tile
								v.PPU.TileData[0][i].Draw(img.img, 0, 0, selectedPalette)
								v.tiles[0][i] = v.PPU.TileData[0][i]

								img.c.Refresh()
							}

							// TODO redraw on open
						} else {
							if !bytes.Equal(v.PPU.TileData[1][i-384][:], v.tiles[1][i-384][:]) {
								// update the tile
								v.PPU.TileData[1][i-384].Draw(img.img, 0, 0, selectedPalette)
								v.tiles[1][i-384] = v.PPU.TileData[1][i-384]

								img.c.Refresh()
							}
						}
					}
				}
			}
		}
	}()

	return nil
}

type tappableImage struct {
	widget.BaseWidget
	c          *canvas.Raster
	img        *image.RGBA
	tapHandler func(event *fyne.PointEvent)
}

func (t *tappableImage) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (t *tappableImage) Tapped(at *fyne.PointEvent) {
	t.tapHandler(at)
}

func (t *tappableImage) TappedSecondary(*fyne.PointEvent) {
	// do nothing
}

func (t *tappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.c)
}

func newTappableImage(img *image.RGBA, c *canvas.Raster, tapHandler func(event *fyne.PointEvent)) *tappableImage {
	t := &tappableImage{img: img, tapHandler: tapHandler, c: c}
	t.ExtendBaseWidget(t)
	return t
}

type customPaddedButton struct {
	widget.BaseWidget

	Button *widget.Button
}

func (c *customPaddedButton) MinSize() fyne.Size {
	return fyne.NewSize(26, 26)
}

func (c *customPaddedButton) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.Button)
}

func newCustomPaddedButton(label string, tapped func()) *customPaddedButton {
	c := &customPaddedButton{}
	c.ExtendBaseWidget(c)
	c.Button = widget.NewButton(label, tapped)
	return c
}

func tileToText(tile ppu.Tile) string {
	var numbers []int
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			var x = i
			var y = j
			high, low := tile[y], tile[y+8]
			var colourNum = int((high >> (7 - x)) & 1)
			colourNum |= int((low>>(7-x))&1) << 1
			numbers = append(numbers, colourNum)
		}
	}

	text := ""
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			text += " " + strconv.Itoa(numbers[j*8+i]) + "  "
		}
		text += "\n\n"
	}

	// remove the last newline
	text = text[:len(text)-1]

	return text
}
