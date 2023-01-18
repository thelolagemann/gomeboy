package cartridge

import (
	"fmt"
)

// MemoryBankedCartridge1 represents a MemoryBankedCartridge1 cartridge. This cartridge type has external RAM and
// supports switching between 2 ROM banks and 4 RAM banks.
type MemoryBankedCartridge1 struct {
	rom     []byte
	romBank uint32

	ram        []byte
	ramBank    uint32
	ramEnabled bool

	romBanking bool

	header *Header
}

// NewMemoryBankedCartridge1 returns a new MemoryBankedCartridge1 cartridge.
func NewMemoryBankedCartridge1(rom []byte, header *Header) *MemoryBankedCartridge1 {
	return &MemoryBankedCartridge1{
		rom:     rom,
		romBank: 1,
		ram:     make([]byte, header.RAMSize),
		header:  header,
	}
}

// Read returns the value from the cartridges ROM or RAM, depending on the bank
// selected.
func (m *MemoryBankedCartridge1) Read(address uint16) uint8 {
	switch {
	case address < 0x4000:
		return m.rom[address] // first bank is always fixed
	case address < 0x8000:
		return m.rom[uint32(address-0x4000)+m.romBank*0x4000] // switchable bank
	case address >= 0xA000 && address < 0xC000:
		if m.ramEnabled {
			return m.ram[uint32(address-0xA000)+m.ramBank*0x2000]
		}
	}

	panic(fmt.Sprintf("invalid address: %X", address))
}

// Write attempts to switch the ROM or RAM bank.
func (m *MemoryBankedCartridge1) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		switch m.header.CartridgeType {
		case MBC1RAM, MBC1RAMBATT:
			m.ramEnabled = value&0x0F == 0x0A
		default:
			return
		}
	case address < 0x4000:
		// ROM bank number (lower 5 bits)
		m.romBank = (m.romBank & 0xE0) | uint32(value&0x1F)
		m.updateRomBank()
	case address < 0x6000:
		if m.romBanking {
			value = (value & 0x03) << 5
			m.romBank = (m.romBank & 0x1F) + uint32(value)
			if m.romBank*0x4000 >= uint32(len(m.rom)) {
				m.romBank = m.romBank % uint32(len(m.rom)/0x4000)
			}
			if m.romBank == 0 {
				m.romBank++
			}
		} else {
			m.ramBank = uint32(value) & 0x03
			if len(m.ram) == 0 {
				m.ramBank = 0
			} else if m.ramBank*0x2000 >= uint32(len(m.ram)) {
				m.ramBank = m.ramBank % uint32(len(m.ram)) / 0x2000
			}
		}
	case address < 0x8000:
		// ROM/RAM mode select
		m.romBanking = value&0x1 == 0x00
	case address >= 0xA000 && address < 0xC000:
		// Write to selected RAM bank
		if m.ramEnabled {
			if m.romBanking {
				m.ram[address&0x1FFF] = value
			} else {
				m.ram[uint16(m.ramBank)*0x2000+address&0x1FFF] = value
			}
		}
	default:
		panic(fmt.Sprintf("mbc1: illegal write to address: %X", address))
	}
}

// updateRomBank updates the romBank if the romBank is out of bounds.
func (m *MemoryBankedCartridge1) updateRomBank() {
	if m.romBank == 0x00 || m.romBank == 0x20 || m.romBank == 0x40 || m.romBank == 0x60 {
		m.romBank++
	}
}

// Save returns the RAM of the cartridge.
func (m *MemoryBankedCartridge1) Save() []byte {
	return m.ram
}

// Load loads the RAM of the cartridge.
func (m *MemoryBankedCartridge1) Load(data []byte) {
	copy(m.ram, data)
}
