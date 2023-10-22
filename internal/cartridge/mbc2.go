package cartridge

import (
	"github.com/thelolagemann/gomeboy/internal/io"
)

// MemoryBankedCartridge2 is a cartridge that supports ROM
// sizes up to 2Mbit (16 banks of 16KiB) and includes an internal
// 512x4 bit RAM array, which is unique amongst MBC cartridges.
type MemoryBankedCartridge2 struct {
	*memoryBankedCartridge
}

// NewMemoryBankedCartridge2 returns a new MemoryBankedCartridge2 cartridge.
func NewMemoryBankedCartridge2(rom []byte, header *Header) *MemoryBankedCartridge2 {
	// override RAM
	header.RAMSize = 2048

	return &MemoryBankedCartridge2{
		memoryBankedCartridge: newMemoryBankedCartridge(rom, header),
	}
}

func (m *MemoryBankedCartridge2) Write(address uint16, value uint8) {
	switch {
	case address <= 0x3FFF:
		if (address & 0x100) == 0x100 {
			m.setROMBank(uint16(value&0x0f), false)

		} else {
			m.ramEnabled = (value & 0x0F) == 0x0A

			if m.ramEnabled {
				m.b.Unlock(io.RAM)
			} else {
				m.b.Lock(io.RAM)
			}
		}
	case address >= 0xA000 && address <= 0xBFFF:
		if m.ramEnabled {
			// make sure to account for ram wrap around by setting the value
			// at 0x200 offsets
			for i := 0; i < 16; i++ {
				m.b.Set(0xA000+(uint16(i)*0x200)+address&0x01ff, value|0xF0)
			}
			m.ram[address&0x01ff] = value | 0xf0
		}
	}
}
