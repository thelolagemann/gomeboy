package cartridge

type MemoryBankedCartridge5 struct {
	*memoryBankedCartridge

	header *Header
}

func NewMemoryBankedCartridge5(rom []byte, header *Header) *MemoryBankedCartridge5 {
	for i := 0xA000; i < 0xC000; i++ {
		header.b.Set(uint16(i), 0xFF) // ram starts disabled
	}
	return &MemoryBankedCartridge5{
		memoryBankedCartridge: newMemoryBankedCartridge(rom, header.RAMSize),
		header:                header,
	}
}

func (m *MemoryBankedCartridge5) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		// RAM enable
		switch m.header.CartridgeType {
		case MBC5RAM, MBC5RAMBATT:
			m.ramEnabled = value&0x0F == 0x0A
		default:
			return
		}
	case address < 0x3000:
		// ROM bank number (lower 8 bits)
		m.romBank = m.romBank&0xFF00 + uint16(value)

		// check if ROM bank has exceeded the number of banks
		if int(m.romBank)*0x4000 >= len(m.rom) {
			m.romBank = (m.romBank) % uint16(len(m.rom)/0x4000)
		}

		// copy data from bank to bus
		m.header.b.CopyTo(0x4000, 0x8000, m.rom[int(m.romBank)*0x4000:])
	case address < 0x4000:
		// ROM bank number (upper 1 bit)
		m.romBank = m.romBank&0x00FF + uint16(value&0x1)<<8

		// check if ROM bank has exceeded the number of banks
		if int(m.romBank)*0x4000 >= len(m.rom) {
			m.romBank = (m.romBank) % uint16(len(m.rom)/0x4000)
		}

		// copy data from bank to bus
		m.header.b.CopyTo(0x4000, 0x8000, m.rom[int(m.romBank)*0x4000:])
	case address < 0x6000:
		// copy data from bus to bank
		m.header.b.CopyFrom(0xA000, 0xC000, m.ram[int(m.ramBank)*0x2000:])

		// RAM bank number
		m.ramBank = (value) & 0xF
		if len(m.ram) <= 0 {
			m.ramBank = 0
		} else if int(m.ramBank)*0x2000 >= len(m.ram) {
			m.ramBank = (m.ramBank) % uint8(len(m.ram)/0x2000)
		}

		// copy data from bank to bus
		m.header.b.CopyTo(0xA000, 0xC000, m.ram[int(m.ramBank)*0x2000:])
	default:
		m.header.b.Set(address, value)
	}
}
