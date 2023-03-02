// Package background provides the background of the Game
// Boy for the PPU.
package background

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types"
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

func (b *Background) init() {
	types.RegisterHardware(
		types.SCX,
		func(v uint8) {
			b.ScrollX = v
		},
		func() uint8 {
			return b.ScrollX
		},
	)
	types.RegisterHardware(
		types.SCY,
		func(v uint8) {
			b.ScrollY = v
		},
		func() uint8 {
			return b.ScrollY
		},
	)
}

// NewBackground returns a new Background.
func NewBackground() *Background {
	b := &Background{}
	b.init()
	return b
}

var _ types.Stater = (*Background)(nil)

func (b *Background) Load(s *types.State) {
	b.ScrollX = s.Read8()
	b.ScrollY = s.Read8()
	b.Palette = palette.LoadPaletteFromState(s)
}

func (b *Background) Save(s *types.State) {
	s.Write8(b.ScrollX)
	s.Write8(b.ScrollY)
	b.Palette.Save(s)
}
