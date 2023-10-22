package cartridge

type MemoryBankedCartridge5 struct {
	*memoryBankedCartridge
}

func NewMemoryBankedCartridge5(rom []byte, header *Header) *MemoryBankedCartridge5 {
	for i := 0xA000; i < 0xC000; i++ {
		header.b.Set(uint16(i), 0xFF) // ram starts disabled
	}
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
		// RAM bank number
		m.ramBank = (value) & 0xF
		if len(m.ram) <= 0 {
			m.ramBank = 0
		} else if int(m.ramBank)*0x2000 >= len(m.ram) {
			m.ramBank = (m.ramBank) % uint8(len(m.ram)/0x2000)
		}

		// copy data from bank to bus
		m.b.CopyTo(0xA000, 0xC000, m.ram[int(m.ramBank)*0x2000:])
	case address >= 0xA000 && address < 0xC000:
		m.b.Set(address, value)
		m.ram[int(m.ramBank)*0x2000+int(address&0x1fff)] = value
	}
}
