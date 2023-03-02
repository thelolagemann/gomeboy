package cartridge

import "github.com/thelolagemann/go-gameboy/internal/types"

// ROMCartridge represents a ROM cartridge. This cartridge type is the simplest
// cartridge type and has no external RAM or MBCm.
type ROMCartridge struct {
	rom []byte
}

func (r *ROMCartridge) Load(s *types.State) {
	// do nothing as ROM is read-only
}

func (r *ROMCartridge) Save(s *types.State) {
	// do nothing as ROM is read-only
}

// NewROMCartridge returns a new ROM cartridge.
func NewROMCartridge(rom []byte) *ROMCartridge {
	return &ROMCartridge{
		rom: rom,
	}
}

// Read returns the value at the given address.
func (r *ROMCartridge) Read(address uint16) uint8 {
	return r.rom[address]
}

// Write writes the value to the given address.
func (r *ROMCartridge) Write(address uint16, value uint8) {
	// do nothing
}
