package ppu

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
)

var (
	_ types.Resettable = &OAM{}
)

// OAM (Object Attribute Memory) is the memory used to store the
// attributes of the sprites. It is 160 bytes long and is located at
// 0xFE00-0xFE9F in the memory map. It is divided in 40 entries of 4 bytes
// each, each entry representing a sprite.
type OAM struct {
	Sprites [40]*Sprite // 40 sprites

	// raw data
	data [160]byte

	highestSprite *Sprite
	lowestSprite  *Sprite
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
	o.lowestSprite = o.Sprites[0]
	o.highestSprite = o.Sprites[len(o.Sprites)-1]
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
	// update sprite
	o.Sprites[address>>2].Update(address, value)

	// update raw data so that it can be easily read back
	o.data[address] = value

	// update highest and lowest y
	if address&3 == 0 {
		// did the sprite just become invisible?
		if value > ScreenHeight {
			// fmt.Println("having to update lowest and highest sprite for", address>>2, value)
			// find the next lowest and highest sprite
			lowestSprite := o.Sprites[0]
			for i := 0; i < len(o.Sprites); i++ {
				if lowestSprite.Y > o.Sprites[i].Y {
					lowestSprite = o.Sprites[i]
				}
			}
			o.lowestSprite = lowestSprite

			highestSprite := o.Sprites[0]
			for i := 0; i < len(o.Sprites); i++ {
				if highestSprite.Y < o.Sprites[i].Y && o.Sprites[i].Y < ScreenHeight {
					highestSprite = o.Sprites[i]
				}
			}
			o.highestSprite = highestSprite
			return // sprite is not visible
		}

		// update lowest and highest y
		if value < o.lowestSprite.Y {
			o.lowestSprite = o.Sprites[address>>2]
		}
		if value > o.highestSprite.Y {
			o.highestSprite = o.Sprites[address>>2]
		}
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
