package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image/color"
	"strconv"
)

var (
	_ display.View = (*PPU)(nil)
)

type PPU struct {
	*ppu.PPU

	dmgPaletteEntryRects    []*canvas.Rectangle
	cgbBgPaletteEntryRects  []*canvas.Rectangle
	cgbObjPaletteEntryRects []*canvas.Rectangle

	grid *fyne.Container
}

func (p *PPU) Run(w fyne.Window, events <-chan display.Event) error {
	// create the base grid and set it as the content of the window
	grid := container.New(layout.NewVBoxLayout())
	w.SetContent(grid)

	// create a grid for the palettes
	dmgPaletteGrid := container.NewGridWithRows(3)
	cgbBgPaletteGrid := container.NewGridWithRows(8)
	cgbObjPaletteGrid := container.NewGridWithRows(8)

	// add the palettes to the grid
	grid.Add(dmgPaletteGrid)
	grid.Add(cgbBgPaletteGrid)
	grid.Add(cgbObjPaletteGrid)

	// create a grid for each entry
	dmgPaletteEntryGrids := make([]*fyne.Container, 3)
	cgbBgPaletteEntryGrids := make([]*fyne.Container, 8)
	cgbObjPaletteEntryGrids := make([]*fyne.Container, 8)

	// dmg palette
	for i, str := range []string{"BG   ", "OBJ 0", "OBJ 1"} {
		dmgPaletteEntryGrids[i] = container.New(&palette{})
		// add the label to the grid
		dmgPaletteEntryGrids[i].Add(widget.NewLabelWithStyle(str, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		dmgPaletteGrid.Add(dmgPaletteEntryGrids[i])
	}
	for i := 0; i < 8; i++ {
		cgbBgPaletteEntryGrids[i] = container.New(&palette{})
		// add the label to the grid
		cgbBgPaletteEntryGrids[i].Add(widget.NewLabelWithStyle("BG  "+strconv.Itoa(i), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		cgbBgPaletteGrid.Add(cgbBgPaletteEntryGrids[i])

		cgbObjPaletteEntryGrids[i] = container.New(&palette{}) // 4 colors + label
		// add the label to the grid
		cgbObjPaletteEntryGrids[i].Add(widget.NewLabelWithStyle("OBJ "+strconv.Itoa(i), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))
		cgbObjPaletteEntryGrids[i].Resize(fyne.NewSize(64, 16))
		cgbObjPaletteGrid.Add(cgbObjPaletteEntryGrids[i])
	}

	// create coloured rectangles for each entry
	dmgPaletteEntryRects := make([]*canvas.Rectangle, 12)
	cgbBgPaletteEntryRects := make([]*canvas.Rectangle, 32)
	cgbObjPaletteEntryRects := make([]*canvas.Rectangle, 32)
	for i := 0; i < 12; i++ {
		dmgPaletteEntryRects[i] = canvas.NewRectangle(color.White)
		dmgPaletteEntryGrids[i/4].Add(dmgPaletteEntryRects[i])
		dmgPaletteEntryRects[i].SetMinSize(fyne.NewSize(24, 24))
	}
	for i := 0; i < 32; i++ {
		cgbBgPaletteEntryRects[i] = canvas.NewRectangle(color.White)
		cgbObjPaletteEntryRects[i] = canvas.NewRectangle(color.White)

		cgbBgPaletteEntryGrids[i/4].Add(cgbBgPaletteEntryRects[i])
		cgbObjPaletteEntryGrids[i/4].Add(cgbObjPaletteEntryRects[i])
		cgbBgPaletteEntryRects[i].SetMinSize(fyne.NewSize(24, 24))
		cgbObjPaletteEntryRects[i].SetMinSize(fyne.NewSize(24, 24))
	}

	// TODO find better way to do this
	// copy the palette entry rectangles to the PPU struct
	p.dmgPaletteEntryRects = dmgPaletteEntryRects
	p.cgbBgPaletteEntryRects = cgbBgPaletteEntryRects
	p.cgbObjPaletteEntryRects = cgbObjPaletteEntryRects

	// set the grid to the PPU struct
	p.grid = grid

	// start the event loop
	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case display.EventTypeQuit:
					return
				case display.EventTypeFrame:
					// set the colors
					for i := uint8(0); i < 12; i++ {
						if i < 4 {
							p.dmgPaletteEntryRects[i].FillColor = toRGB(p.PPU.Palette.GetColour(i % 4))
						} else if i < 8 {
							p.dmgPaletteEntryRects[i].FillColor = toRGB(p.PPU.SpritePalettes[0].GetColour(i % 4))
						} else {
							p.dmgPaletteEntryRects[i].FillColor = toRGB(p.PPU.SpritePalettes[1].GetColour(i % 4))
						}
						p.dmgPaletteEntryRects[i].Refresh()
					}
					for i := uint8(0); i < 32; i++ {
						p.cgbBgPaletteEntryRects[i].FillColor = toRGB(p.PPU.ColourPalette.GetColour(i/4, i%4))
						p.cgbObjPaletteEntryRects[i].FillColor = toRGB(p.PPU.ColourSpritePalette.GetColour(i/4, i%4))
						p.cgbBgPaletteEntryRects[i].Refresh()
						p.cgbObjPaletteEntryRects[i].Refresh()
					}

				}
			}
		}
	}()

	return nil
}

func NewPPU(ppu *ppu.PPU) *PPU {
	return &PPU{PPU: ppu}
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

type palette struct {
}

func (p *palette) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(164, 24)
}

func (p *palette) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
	pos := fyne.NewPos(0, 0)
	for _, o := range objects {
		s := o.MinSize()
		o.Resize(s)
		o.Move(pos)

		pos = pos.Add(fyne.NewPos(s.Width+4, 0))
	}
}
