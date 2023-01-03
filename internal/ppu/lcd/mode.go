package lcd

// Mode represents a mode of the LCD.
type Mode = int

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
