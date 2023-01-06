package cartridge

import "github.com/thelolagemann/go-gameboy/internal/ram"

// MemoryBankController represents a Memory Bank Controller.
// It provides an interface for reading and writing to the cartridge,
// which provides a unified interface for all cartridge types. The
// cartridge implementations are responsible for handling the
// memory bank switching.
type MemoryBankController interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)
}

// MemoryBankedCartridge represents a MemoryBankedCartridge.
type MemoryBankedCartridge struct {
	rom     []byte
	romBank uint8

	ram        ram.RAM
	ramBank    uint8
	ramEnabled bool

	romBanking bool

	MemoryBankController
}

func (m *MemoryBankedCartridge) Header() Header {
	//TODO implement me
	panic("implement me")
}

func (m *MemoryBankedCartridge) Title() string {
	return ""
}

// NewMBC1 returns a new MemoryBankedCartridge.
func NewMBC1(rom []byte) *MemoryBankedCartridge {
	return &MemoryBankedCartridge{
		rom: rom,
	}
}

// Read returns the value at the given address.
func (m *MemoryBankedCartridge) Read(address uint16) uint8 {
	switch {
	case address < 0x4000:
		return m.rom[address] // ROM bank 0
	case address < 0x8000:
		return m.rom[(address-0x4000)+uint16(m.romBank)*0x4000] // ROM bank 1-127
	default:
		return m.ram.Read((address - 0xA000) + uint16(m.ramBank)*0x2000) // RAM bank 0-3
	}
}

// Write writes the value to the given address.
func (m *MemoryBankedCartridge) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		// RAM enable
		if value&0xF == 0xA {
			m.ramEnabled = true
		} else {
			m.ramEnabled = false
		}
	case address < 0x4000:
		// ROM bank number
		m.romBank = (m.romBank & 0xE0) | (value & 0x1F)
		m.updateROMBank()
	case address < 0x6000:
		// ROM/RAM bank select
		if m.romBanking {
			m.romBank = (m.romBank & 0x1F) | (value & 0xE0)
			m.updateROMBank()
		} else {
			m.ramBank = value & 0x3
		}
	case address < 0x8000:
		// ROM/RAM mode select
		m.romBanking = value&1 == 0
		if m.romBanking {
			m.ramBank = 0
		} else {
			m.romBank = m.romBank & 0x1F
		}
	default:
		// RAM
		if m.ramEnabled {
			m.ram.Write((address-0xA000)+uint16(m.ramBank)*0x2000, value)
		}
	}
}

// updateROMBank updates the ROM bank.
func (m *MemoryBankedCartridge) updateROMBank() {
	if m.romBank == 0x00 || m.romBank == 0x20 || m.romBank == 0x40 || m.romBank == 0x60 {
		m.romBank++
	}
}
