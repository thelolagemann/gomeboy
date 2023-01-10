package cartridge

// ROMCartridge represents a ROM cartridge. This cartridge type is the simplest
// cartridge type and has no external RAM or MBCm.
type ROMCartridge struct {
	rom []byte
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
