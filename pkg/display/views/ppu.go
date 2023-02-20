package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"image/color"
	"strconv"
	"time"
)

type PPU struct {
	*ppu.PPU
}

func NewPPU(ppu *ppu.PPU) *PPU {
	return &PPU{ppu}
}

func (p *PPU) Run(w fyne.Window) error {
	// create the base grid and set it as the content of the window
	grid := container.New(layout.NewVBoxLayout())
	w.SetContent(grid)

	// create a grid for the DMG palettes
	dmgPaletteGrid := container.NewGridWithRows(3)

	// create a grid for the CGB BG palette
	cgbBgPaletteGrid := container.NewGridWithRows(8)

	// create a grid for the CGB OBJ palette
	cgbObjPaletteGrid := container.NewGridWithRows(8)

	// add the palette grids to the palette grid
	grid.Add(dmgPaletteGrid)
	grid.Add(cgbBgPaletteGrid)
	grid.Add(cgbObjPaletteGrid)

	// create a grid for each entry
	dmgPaletteEntryGrids := make([]*fyne.Container, 3)
	cgbBgPaletteEntryGrids := make([]*fyne.Container, 8)
	cgbObjPaletteEntryGrids := make([]*fyne.Container, 8)

	// dmg palette
	for i, str := range []string{"BG   ", "OBJ 0", "OBJ 1"} {
		dmgPaletteEntryGrids[i] = container.New(layout.NewHBoxLayout())
		// add the label to the grid
		dmgPaletteEntryGrids[i].Add(widget.NewLabelWithStyle(str, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		dmgPaletteGrid.Add(dmgPaletteEntryGrids[i])
	}
	for i := 0; i < 8; i++ {
		cgbBgPaletteEntryGrids[i] = container.New(layout.NewHBoxLayout())
		// add the label to the grid
		cgbBgPaletteEntryGrids[i].Add(widget.NewLabelWithStyle("BG  "+strconv.Itoa(i), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		cgbBgPaletteGrid.Add(cgbBgPaletteEntryGrids[i])

		cgbObjPaletteEntryGrids[i] = container.NewHBox() // 4 colors + label
		// add the label to the grid
		cgbObjPaletteEntryGrids[i].Add(widget.NewLabelWithStyle("OBJ "+strconv.Itoa(i), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		cgbObjPaletteGrid.Add(cgbObjPaletteEntryGrids[i])
	}

	// create coloured rectangles for each entry
	dmgPaletteEntryRects := make([]*canvas.Rectangle, 12)
	cgbBgPaletteEntryRects := make([]*canvas.Rectangle, 32)
	cgbObjPaletteEntryRects := make([]*canvas.Rectangle, 32)
	for i := 0; i < 12; i++ {
		dmgPaletteEntryRects[i] = canvas.NewRectangle(color.White)
		dmgPaletteEntryGrids[i/4].Add(dmgPaletteEntryRects[i])
		dmgPaletteEntryRects[i].SetMinSize(fyne.NewSize(32, 32))
	}
	for i := 0; i < 32; i++ {
		cgbBgPaletteEntryRects[i] = canvas.NewRectangle(color.White)
		cgbObjPaletteEntryRects[i] = canvas.NewRectangle(color.White)

		cgbBgPaletteEntryGrids[i/4].Add(cgbBgPaletteEntryRects[i])
		cgbObjPaletteEntryGrids[i/4].Add(cgbObjPaletteEntryRects[i])
		cgbBgPaletteEntryRects[i].SetMinSize(fyne.NewSize(32, 32))
		cgbObjPaletteEntryRects[i].SetMinSize(fyne.NewSize(32, 32))
	}

	// create a goroutine to update the palette every 100ms
	go func() {
		for {
			// set the colors
			for i := uint8(0); i < 12; i++ {
				dmgPaletteEntryRects[i].FillColor = toRGB(p.PPU.Palette.GetColour(i % 4))
			}
			for i := uint8(0); i < 32; i++ {
				cgbBgPaletteEntryRects[i].FillColor = toRGB(p.PPU.ColourPalette.GetColour(i/4, i%4))
				cgbObjPaletteEntryRects[i].FillColor = toRGB(p.PPU.ColourSpritePalette.GetColour(i/4, i%4))
			}
			time.Sleep(10 * time.Millisecond)

			grid.Refresh()
		}
	}()

	return nil
}

// toRGB converts a 3 element uint8 array to a color.RGBA
// with an alpha value of 255 (opaque)
func toRGB(rgb [3]uint8) color.RGBA {
	return color.RGBA{rgb[0], rgb[1], rgb[2], 255}
}

// TODO
// - create function to create palette grid (colours 0 - 3)
// - new window function
// - window interface - Run() error - creates a new window and runs it, Update() error - updates the window when appropriate
// - channel from main window that sends a signal over channel to update all windows on new frame
// - palettes actually hold colours (not just indexes) - palette changes
