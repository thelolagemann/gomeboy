// Package background provides the background of the Game
// Boy for the PPU.
package background

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
)

// Background represents the background. It is made up of a 256x256 pixel map
// of tiles. The map is divided into 32x32 tiles. Each tile is 8x8 pixels. As the
// display only has 160x144 pixels, the background is scrolled to display
// different parts of the map.
type Background struct {
	// ScrollY is the Y position of the background.
	ScrollY uint8
	// ScrollX is the X position of the background.
	ScrollX uint8
	// Palette is the current background palette.
	Palette palette.Palette
}

// NewBackground returns a new Background.
func NewBackground() *Background {
	return &Background{
		ScrollY: 0,
		ScrollX: 0,
	}
}

// Read reads a byte from the background.
func (b *Background) Read(addr uint16) uint8 {
	switch addr {
	case registers.SCY:
		return b.ScrollY
	case registers.SCX:
		return b.ScrollX
	case registers.BGP:
		return b.Palette.ToByte()
	}

	panic(fmt.Sprintf("background: illegal read from address 0x%04X", addr))
}

// Write writes a byte to the background.
func (b *Background) Write(addr uint16, val uint8) {
	switch addr {
	case registers.SCY:
		b.ScrollY = val
	case registers.SCX:
		b.ScrollX = val
	case registers.BGP:
		b.Palette = palette.ByteToPalette(val)
	default:
		panic(fmt.Sprintf("background: illegal write to address 0x%04X", addr))
	}
}
