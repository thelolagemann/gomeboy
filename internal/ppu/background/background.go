// Package background provides the background of the Game
// Boy for the PPU.
package background

import (
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

// Background represents the background.
type Background struct {
	ScrollY uint8

	ScrollX uint8
	// Palette is the current background palette.
	Palette palette.Palette
}

func (b *Background) init() {

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
