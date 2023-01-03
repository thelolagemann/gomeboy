package cartridge

// ROMCartridge represents a ROM cartridge. This cartridge type is the simplest
// cartridge type and has no external RAM or MBC.
type ROMCartridge struct {
	baseCartridge
}

// NewROMCartridge returns a new ROM cartridge.
func NewROMCartridge(rom []byte) *ROMCartridge {
	return &ROMCartridge{
		baseCartridge: baseCartridge{
			rom:    rom,
			header: parseHeader(rom[0x100:0x14F]),
		},
	}
}