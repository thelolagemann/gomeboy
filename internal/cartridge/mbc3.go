package cartridge

import "github.com/thelolagemann/go-gameboy/internal/ram"

// MemoryBankedCartridge3 represents a MemoryBankedCartridge3 cartridge. This cartridge type has external RAM and
// supports switching between 2 ROM banks and 4 RAM banks, and provides a real time clock.
type MemoryBankedCartridge3 struct {
	rom     []byte
	romBank uint32

	ram        ram.RAM
	ramBank    uint8
	ramEnabled bool

	rtc        []byte
	latchedRTC []byte
	latched    bool
}

// NewMemoryBankedCartridge3 returns a new MemoryBankedCartridge3 cartridge.
func NewMemoryBankedCartridge3(rom []byte) *MemoryBankedCartridge3 {
	return &MemoryBankedCartridge3{
		rom:        rom,
		romBank:    1,
		ram:        ram.NewRAM(0x8000),
		rtc:        make([]byte, 0x10),
		latchedRTC: make([]byte, 0x10),
	}
}

// Read returns the value from the cartridges ROM or RAM, depending on the bank
// selected.
func (m *MemoryBankedCartridge3) Read(address uint16) uint8 {
	switch {
	case address < 0x4000:
		return m.rom[address]
	case address < 0x8000:
		return m.rom[uint16(m.romBank)*0x4000+address-0x4000]
	case address >= 0xA000 && address < 0xC000:
		if m.ramBank >= 0x4 {
			if m.latched {
				return m.latchedRTC[m.ramBank-0x4]
			}
			return m.rtc[m.ramBank-0x4]
		}
		return m.ram.Read(uint16(m.ramBank)*0x2000 + address - 0xA000)
	}

	return 0xFF
}

// Write attempts to switch the ROM or RAM bank.
func (m *MemoryBankedCartridge3) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		// RAM enable
		m.ramEnabled = (value & 0xA) != 0
	case address < 0x4000:
		// ROM bank number (lower 5 bits)
		m.romBank = uint32(value & 0x7F)
		// TODO update romBank if romBank == 0
	case address < 0x6000:
		// RAM bank number or upper ROM bank number
		m.ramBank = value
	case address < 0x8000:
		if value == 0x1 {
			m.latched = false
		} else if value == 0x0 {
			m.latched = true
			copy(m.rtc, m.latchedRTC)
		}

	case address >= 0xA000 && address < 0xC000:
		if m.ramBank >= 0x4 {
			m.rtc[m.romBank] = value
		} else {
			m.ram.Write(uint16(m.ramBank)*0x2000+address-0xA000, value)
		}
	}
}
