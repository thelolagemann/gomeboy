// Package background provides the background of the Game
// Boy for the PPU.
package background

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
)

// Background represents the background. It is made up of a 256x256 pixel map
// of tiles. The map is divided into 32x32 tiles. Each tile is 8x8 pixels. As the
// display only has 160x144 pixels, the background is scrolled to display
// different parts of the map.
type Background struct {
	// ScrollY is the Y position of the background.
	ScrollY *registers.Hardware
	// ScrollX is the X position of the background.
	ScrollX *registers.Hardware
	// Palette is the current background palette.
	palette *registers.Hardware
}

func (b *Background) init() {
	// setup the registers
	b.ScrollY = registers.NewHardware(registers.SCY, registers.IsReadableWritable())
	b.ScrollX = registers.NewHardware(registers.SCX, registers.IsReadableWritable())
	b.palette = registers.NewHardware(registers.BGP, registers.IsReadableWritable())
}

// NewBackground returns a new Background.
func NewBackground() *Background {
	b := &Background{}
	b.init()
	return b
}

func (b *Background) Palette() palette.Palette {
	return palette.ByteToPalette(b.palette.Value()) // TODO handle CGB
}
