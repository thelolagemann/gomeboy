package ppu

import (
	"github.com/thelolagemann/gomeboy/internal/types"
)

var (
	_ types.Resettable = &OAM{}
)

// OAM (Object Attribute Memory) is the memory used to store the
// Attributes of the sprites. It is 160 bytes long and is located at
// 0xFE00-0xFE9F in the memory map. It is divided in 40 entries of 4 bytes
// each, each entry representing a sprite.
type OAM struct {
	Sprites [40]*Sprite // 40 sprites

	// raw data
	data [160]byte

	dirtyScanlines        [ScreenHeight]bool
	spriteScanlines       [ScreenHeight]bool
	spriteScanlinesColumn [ScreenHeight][ScreenWidth]bool
}

// Reset implements the types.Resettable interface.
func (o *OAM) Reset() {
	// setup sprites
	for i := len(o.Sprites) - 1; i >= 0; i-- {
		o.Sprites[i] = &Sprite{
			spriteAttributes: spriteAttributes{},
		}
	}
	// reset raw data
	o.data = [160]byte{}
}

func NewOAM() *OAM {
	o := &OAM{}
	o.Reset()
	return o
}

// Read returns the value at the given address.
func (o *OAM) Read(address uint16) uint8 {
	return o.data[address]
}

// Write writes the given value at the given address.
func (o *OAM) Write(address uint16, value uint8) {
	// check if the address is valid
	// get the sprite
	s := o.Sprites[address>>2]

	// update raw data so that it can be easily read back
	o.data[address] = value

	oldY := s.Y
	oldX := s.X

	// update the sprite Attributes
	byteIndex := address % 4
	if byteIndex == 0 {
		s.Y = value - 16

		// was the s visible before?
		if oldY < ScreenHeight && oldX < ScreenWidth {
			// we need to remove the positions that the s was visible on
			for i := oldY; i < oldY+8 && i < ScreenHeight; i++ {
				o.spriteScanlines[i] = false
				o.dirtyScanlines[i] = true
				for j := oldX; j < oldX+8 && j < ScreenWidth; j++ {
					o.spriteScanlinesColumn[i][j] = false
				}
			}
		}

		// is the s visible now?
		newYPos := s.Y
		if newYPos > ScreenHeight || oldX > ScreenHeight {
			return // s is not visible
		}

		// we need to add the positions that the s is now visible on
		for i := newYPos; i < newYPos+8 && i < ScreenHeight; i++ {
			o.spriteScanlines[i] = true
			for j := oldX; j < oldX+8 && j < ScreenWidth; j++ {
				o.spriteScanlinesColumn[i][j] = true
			}
		}
	} else if byteIndex == 1 {
		s.X = value - 8
		// was the s visible before?
		if oldY < ScreenHeight && oldX < ScreenWidth {
			// we need to remove the positions that the s was visible on
			for i := oldY; i < oldY+8 && i < ScreenHeight; i++ {
				o.spriteScanlines[i] = false
				o.dirtyScanlines[i] = true
				for j := oldX; j < oldX+8 && j < ScreenWidth; j++ {
					o.spriteScanlinesColumn[i][j] = false
				}
			}
		}

		// is the s visible now?
		newXPos := s.X
		if newXPos > ScreenWidth || oldY > ScreenHeight {
			return // s is not visible
		}

		// we need to add the positions that the s is now visible on
		for i := oldY; i < oldY+8 && i < ScreenHeight; i++ {
			o.spriteScanlines[i] = true
			for j := newXPos; j < newXPos+8 && j < ScreenWidth; j++ {
				o.spriteScanlinesColumn[i][j] = true
			}
		}
	} else if byteIndex == 2 {
		s.TileID = value
	} else if byteIndex == 3 {
		s.priority = value&0x80 == 0
		s.flipY = value&0x40 != 0
		s.flipX = value&0x20 != 0
		s.useSecondPalette = value & 0x10 >> 4
		s.vRAMBank = (value >> 3) & 0x01
		s.cgbPalette = value & 0x07
	}

}

var _ types.Stater = (*OAM)(nil)

func (o *OAM) Load(s *types.State) {
	s.ReadData(o.data[:])
	for i := 0; i < len(o.data); i++ {
		o.Write(uint16(i), o.data[i])
	}
}

func (o *OAM) Save(s *types.State) {
	s.WriteData(o.data[:])
}
