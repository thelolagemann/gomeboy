package lcd

// Mode represents a mode of the LCD.
type Mode uint8

const (
	// HBlank is the horizontal blanking mode. The CPU can access both the display RAM and OAM.
	HBlank Mode = iota
	// VBlank is the vertical blanking mode. The CPU can access both the display RAM and OAM.
	VBlank
	// OAM is the OAM mode. The CPU can access OAM but not the display RAM.
	OAM
	// VRAM is the VRAM mode. The CPU can access the display RAM but not OAM.
	VRAM
)

var scrollXDots = []uint16{
	0, 4, 4, 4, 4, 8, 8, 8,
}

func (m Mode) Dots(scrollX uint8) uint16 {
	scrollAdjusted := scrollX % 0x08
	switch m {
	case HBlank:
		return 200 - scrollXDots[scrollAdjusted]
	case VBlank:
		return 456
	case OAM:
		return 84
	case VRAM:
		return 172 + scrollXDots[scrollAdjusted]
	}
	return 0
}

func (m Mode) AdjustedDots(scrollX uint8) uint16 {
	scrollAdjusted := scrollX % 0x08
	switch m {
	case HBlank:
		return scrollXDots[scrollAdjusted]
	case VBlank:
		return 456
	case OAM:
		return 84
	case VRAM:
		return 172
	}
	return 0
}
