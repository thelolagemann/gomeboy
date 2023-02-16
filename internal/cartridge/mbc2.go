package cartridge

type MemoryBankedCartridge2 struct {
	rom    []byte
	ram    []byte
	header *Header

	ramg bool
	romb uint8
}

// NewMemoryBankedCartridge2 returns a new MemoryBankedCartridge2 cartridge.
func NewMemoryBankedCartridge2(rom []byte, header *Header) *MemoryBankedCartridge2 {
	return &MemoryBankedCartridge2{
		rom:    rom,
		ram:    make([]byte, 512),
		header: header,
		romb:   0x01,
	}
}

// Read returns the value from the cartridges ROM or RAM, depending on the bank
// selected.
func (m *MemoryBankedCartridge2) Read(address uint16) uint8 {
	if address <= 0x3FFF {
		return m.rom[address]
	} else if address <= 0x7FFF {
		offset := uint32(m.romb) * 0x4000
		if offset >= uint32(len(m.rom)) {
			offset = offset % uint32(len(m.rom))
		}
		return m.rom[offset+uint32(address-0x4000)]
	} else if address >= 0xA000 && address < 0xC000 {
		if !m.ramg {
			return 0xFF
		}
		return m.ram[address&0x01FF] | 0xF0
	} else {
		return 0xFF
	}
}

func (m *MemoryBankedCartridge2) Write(address uint16, value uint8) {
	if address <= 0x3FFF {
		if (address & 0x100) == 0x100 {
			m.romb = value & 0x0F
			if m.romb == 0 {
				m.romb = 1
			}
		} else {
			m.ramg = (value & 0x0F) == 0x0A
		}
	} else if address >= 0xA000 && address < 0xC000 {
		if len(m.ram) == 0 || !m.ramg {
			return
		}
		m.ram[address-0xA000] = value
	}
}
