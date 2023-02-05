package ppu

// Sprite is the 8x8 default sprite.
type Sprite struct {
	X      uint8
	Y      uint8
	TileID uint8
	*SpriteAttributes
}

// SpriteAttributes represents the attributes of a sprite.
type SpriteAttributes struct {
	// Bit 7 - OBJ-to-BG Priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3)
	// (Used for both BG and Window. BG color 0 is always behind OBJ)
	Priority bool
	// Bit 6 - Y flip          (0=Normal, 1=Vertically mirrored)
	FlipY bool
	// Bit 5 - X flip          (0=Normal, 1=Horizontally mirrored)
	FlipX bool
	// Bit 4 - Palette number  **Non CGB Mode Only** (0=OBP0, 1=OBP1)
	UseSecondPalette uint8
	// Bit 3 - Tile VRAM-Bank  **CGB Mode Only**     (0=Bank 0, 1=Bank 1)
	VRAMBank uint8
	// Bit 0-2 - Palette number  **CGB Mode Only**     (OBP0-7)
	CGBPalette uint8
}

// Read returns the value of the sprite attributes.
func (sa *SpriteAttributes) Read() uint8 {
	var value uint8
	if sa.Priority {
		value |= 1 << 7
	}
	if sa.FlipY {
		value |= 1 << 6
	}
	if sa.FlipX {
		value |= 1 << 5
	}
	if sa.UseSecondPalette == 1 {
		value |= 1 << 4
	}
	value |= sa.VRAMBank << 3
	value |= sa.CGBPalette
	return value
}

func (s *Sprite) Update(address uint16, value uint8) {
	switch address % 4 {
	case 0:
		s.Y = value
	case 1:
		s.X = value
	case 2:
		s.TileID = value
	case 3:
		s.Priority = value&0x80 == 0
		s.FlipY = value&0x40 != 0
		s.FlipX = value&0x20 != 0
		if value&0x10 != 0 {
			s.UseSecondPalette = 1
		} else {
			s.UseSecondPalette = 0
		}
		s.VRAMBank = (value >> 3) & 0x01
		s.CGBPalette = value & 0x07
	}
}

// Read returns the value of the sprite at the given address.
func (s *Sprite) Read(address uint16) uint8 {
	switch address & 0b11 {
	case 0:
		return s.Y
	case 1:
		return s.X
	case 2:
		return s.TileID
	case 3:
		return s.SpriteAttributes.Read()
	default:
		return 0xFF
	}
}

func (s *Sprite) GetX() uint8 {
	return s.X - 8
}

func (s *Sprite) GetY() uint8 {

	return s.Y - 16
}
