package ppu

// OAM (Object Attribute Memory) is the memory used to store the
// attributes of the sprites. It is 160 bytes long and is located at
// 0xFE00-0xFE9F in the memory map. It is divided in 40 entries of 4 bytes
// each, each entry representing a sprite.
type OAM struct {
	Sprites [40]*Sprite // 40 sprites

	// raw data
	data [160]byte
}

func (o *OAM) init() {
	// setup sprites
	for i := len(o.Sprites) - 1; i >= 0; i-- {
		o.Sprites[i] = &Sprite{
			SpriteAttributes: SpriteAttributes{},
		}
	}
}

func NewOAM() *OAM {
	o := &OAM{
		data: [160]byte{},
	}
	o.init()
	return o
}

// Read returns the value at the given address.
func (o *OAM) Read(address uint16) uint8 {
	return o.data[address]
}

// Write writes the given value at the given address.
func (o *OAM) Write(address uint16, value uint8) {
	// get the sprite index
	o.Sprites[address>>2].Update(address, value)

	// update raw data so that it can be easily read back
	o.data[address] = value
}