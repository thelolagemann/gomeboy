package ppu

// Sprite represents a sprite. Nintendo refers to them as "OBJs", but we'll
// stick with the more common "sprite" name.
type Sprite struct {
	// Y position of the sprite. The sprite's vertical position on the screen
	// + 16.
	Y int8
	// X position of the sprite. The sprite's horizontal position on the screen
	// + 8.
	X    int8
	Tile uint8
	SpriteAttributes
}

func NewSprite(b []byte) Sprite {
	return Sprite{
		Y:                int8(b[0]) - 16,
		X:                int8(b[1]) - 8,
		Tile:             b[2],
		SpriteAttributes: NewSpriteAttributes(b[3]),
	}
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
	UseSecondPalette bool
	// Bit 3 - Tile VRAM-Bank  **CGB Mode Only**     (0=Bank 0, 1=Bank 1)
	VRAMBank uint8
	// Bit 0-2 - Palette number  **CGB Mode Only**     (OBP0-7)
	CGBPalette uint8
}

func NewSpriteAttributes(b uint8) SpriteAttributes {
	return SpriteAttributes{
		Priority:         b&0x80 != 0,
		FlipY:            b&0x40 != 0,
		FlipX:            b&0x20 != 0,
		UseSecondPalette: b&0x10 != 0,
		VRAMBank:         b & 0x08,
		CGBPalette:       b & 0x07,
	}
}
