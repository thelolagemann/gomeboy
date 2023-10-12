package cartridge

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/types"
)

type MemoryBankedCartridge5 struct {
	rom []byte
	ram []byte

	ramEnabled bool
	romBank    uint16
	ramBank    uint8

	header *Header
}

func (m *MemoryBankedCartridge5) Load(s *types.State) {
	m.ramEnabled = s.ReadBool()
	m.romBank = s.Read16()
	m.ramBank = s.Read8()
}

func (m *MemoryBankedCartridge5) Save(s *types.State) {
	s.WriteBool(m.ramEnabled)
	s.Write16(m.romBank)
	s.Write8(m.ramBank)
}

func NewMemoryBankedCartridge5(rom []byte, header *Header) *MemoryBankedCartridge5 {
	return &MemoryBankedCartridge5{
		rom:     rom,
		header:  header,
		romBank: 1,
		ram:     make([]byte, header.RAMSize),
	}
}

func (m *MemoryBankedCartridge5) Read(address uint16) uint8 {
	switch {
	case address < 0x4000:
		return m.rom[address] // first bank is always fixed
	case address < 0x8000:
		return m.rom[m.romBank*0x4000+int(address&0x3FFF)] // switchable bank
	case address >= 0xA000 && address < 0xC000:
		if m.ramEnabled {
			return m.ram[m.ramBank*0x2000+int(address&0x1FFF)]
		} else {
			return 0xFF
		}
	}

	panic(fmt.Sprintf("invalid address: %X", address))
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
		// copy data from bus to existing bank

		// ROM bank number (lower 8 bits)
		m.romBank = (m.romBank)&0xFF00 + uint16(value)

		// check if ROM bank has exceeded the number of banks
		if int(m.romBank*0x4000) >= len(m.rom) {
			m.romBank = (m.romBank) % uint16(len(m.rom)/0x4000)
		}

		// copy data from bank to bus
	case address < 0x4000:
		// ROM bank number (upper 1 bit)
		m.romBank = (m.romBank & 0x00FF) + ((uint16(value) & 0x1) << 8)

		// check if ROM bank has exceeded the number of banks
		if int(m.romBank*0x4000) >= len(m.rom) {
			m.romBank = (m.romBank) % uint16(len(m.rom)/0x4000)
		}

		// copy data from bank to bus
	case address < 0x6000:
		// copy data from bus to bank

		// RAM bank number
		m.ramBank = (value) & 0xF
		if len(m.ram) <= 0 {
			m.ramBank = 0
		} else if int(m.ramBank)*0x2000 >= len(m.ram) {
			m.ramBank = (m.ramBank) % uint8(len(m.ram)/0x2000)
		}
	}
}

func (m *MemoryBankedCartridge5) LoadRAM(data []byte) {
	copy(m.ram, data)
}

func (m *MemoryBankedCartridge5) SaveRAM() []byte {
	return m.ram
}
