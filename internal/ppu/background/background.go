// Package background provides the background of the Game
// Boy for the PPU.
package background

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
)

const (
	// ScrollYRegister is the address of the backgrounds scroll Y register.
	// This register is used to scroll the background up and down.
	ScrollYRegister = 0xFF42
	// ScrollXRegister is the address of the backgrounds scroll X register.
	// This register is used to scroll the background left and right.
	ScrollXRegister = 0xFF43
	// PaletteRegister is the address of the background palette register.
	// This register contains the colour palette for the background, and
	// is only used in DMG mode. In CGB mode, the background palette is
	// stored in the CGB's palette RAM.
	//
	// The palette is stored in the following format:
	//   Bit 7-6 - Shade for Color Number 3
	//   Bit 5-4 - Shade for Color Number 2
	//   Bit 3-2 - Shade for Color Number 1
	//   Bit 1-0 - Shade for Color Number 0
	PaletteRegister = 0xFF47
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
	case ScrollYRegister:
		return b.ScrollY
	case ScrollXRegister:
		return b.ScrollX
	case PaletteRegister:
		return b.Palette.ToByte()
	}

	panic(fmt.Sprintf("background: illegal read from address 0x%04X", addr))
}

// Write writes a byte to the background.
func (b *Background) Write(addr uint16, val uint8) {
	switch addr {
	case ScrollYRegister:
		b.ScrollY = val
	case ScrollXRegister:
		b.ScrollX = val
	case PaletteRegister:
		b.Palette = palette.ByteToPalette(val)
	default:
		panic(fmt.Sprintf("background: illegal write to address 0x%04X", addr))
	}
}
