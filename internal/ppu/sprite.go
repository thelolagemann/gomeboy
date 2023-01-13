package ppu

// Sprite is the 8x8 default sprite.
type Sprite struct {
	*SpriteAttributes
	tileID          int
	CurrentTileLine int
}

func NewSprite() Sprite {
	var sprite Sprite
	sprite.SpriteAttributes = &SpriteAttributes{}
	return sprite
}

func (s *Sprite) UpdateSprite(address uint16, value uint8) {
	var attrId = int(address) % 4
	if attrId == 2 {
		s.tileID = int(value)
	} else {
		s.SpriteAttributes.Update(attrId, value)
	}
}

// SpriteAttributes represents the attributes of a sprite.
type SpriteAttributes struct {
	X    uint8
	Y    uint8
	Tile uint8
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

func (s *SpriteAttributes) Update(attribute int, value uint8) {
	switch attribute {
	case 0:
		s.Y = value - 16
	case 1:
		s.X = value - 8
	case 2:
		s.Tile = value
	case 3:
		s.Priority = value&0x80 == 0
		s.FlipY = value&0x40 != 0
		s.FlipX = value&0x20 != 0
		if value&0x10 != 0 {
			s.UseSecondPalette = 1
		} else {
			s.UseSecondPalette = 0
		}
		s.VRAMBank = value & 0x08
		s.CGBPalette = value & 0x07
	}
}
