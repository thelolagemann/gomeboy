package cartridge

type MemoryBankedCartridge5 struct {
	*memoryBankedCartridge
}

func NewMemoryBankedCartridge5(rom []byte, header *Header) *MemoryBankedCartridge5 {
	return &MemoryBankedCartridge5{
		memoryBankedCartridge: newMemoryBankedCartridge(rom, header),
	}
}

func (m *MemoryBankedCartridge5) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		// RAM enable
		switch m.CartridgeType {
		case MBC5RAM, MBC5RAMBATT:
			m.ramEnabled = value&0x0F == 0x0A
		}
	case address < 0x3000:
		m.setROMBank(m.romBank&0xFF00+uint16(value), true)
	case address < 0x4000:
		m.setROMBank(m.romBank&0x00FF+uint16(value&0x1)<<8, true)
	case address < 0x6000:
		m.setRAMBank(value & 0x0f)
	case address >= 0xA000 && address < 0xC000:
		m.b.Set(address, value)
		m.ram[int(m.ramBank)*0x2000+int(address&0x1fff)] = value
	}
}
