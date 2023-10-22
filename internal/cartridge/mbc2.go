package cartridge

import (
	"github.com/thelolagemann/gomeboy/internal/io"
)

// MemoryBankedCartridge2 is a cartridge that supports ROM
// sizes up to 2Mbit (16 banks of 16KiB) and includes an internal
// 512x4 bit RAM array, which is unique amongst MBC cartridges.
type MemoryBankedCartridge2 struct {
	header *Header
	*memoryBankedCartridge
}

// NewMemoryBankedCartridge2 returns a new MemoryBankedCartridge2 cartridge.
func NewMemoryBankedCartridge2(rom []byte, header *Header) *MemoryBankedCartridge2 {
	header.b.Lock(io.RAM)
	return &MemoryBankedCartridge2{
		memoryBankedCartridge: newMemoryBankedCartridge(rom, 512),
		header:                header,
	}
}

func (m *MemoryBankedCartridge2) Write(address uint16, value uint8) {
	switch {
	case address <= 0x3FFF:
		if (address & 0x100) == 0x100 {
			m.romBank = uint16(value & 0x0F)
			if m.romBank == 0 {
				m.romBank = 1
			}

			// check to see if banks exceed rom
			if int(m.romBank)*0x4000 >= len(m.rom) {
				m.romBank = m.romBank % uint16(len(m.rom)/0x4000)
			}

			// copy from bank to bus
			m.header.b.CopyTo(0x4000, 0x8000, m.rom[int(m.romBank)*0x4000:])
		} else {
			if m.ramEnabled {
				// copy data from bus to ram
				m.header.b.CopyFrom(0xA000, 0xA200, m.ram)
			}
			m.ramEnabled = (value & 0x0F) == 0x0A

			if m.ramEnabled {
				// only 2048 bits, so we need to account for RAM wrap around
				// TODO handle on RAM write
				for i := 0; i < 16; i++ {
					m.header.b.CopyTo(0xA000+(uint16(i)*0x200), 0xA200+(uint16(i)*0x200), m.ram)
				}
				m.header.b.Unlock(io.RAM)
			} else {
				m.header.b.Lock(io.RAM)
			}
		}
	case address >= 0xA000 && address <= 0xBFFF:
		// make sure to account for ram wrap around by setting the
		if m.ramEnabled {
			for i := 0; i < 16; i++ {
				m.header.b.Set(0xA000+(uint16(i)*0x200)+address&0x01ff, value|0xF0)
			}
		}
	}

}
