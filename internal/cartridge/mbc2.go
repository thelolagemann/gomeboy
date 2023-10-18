package cartridge

import (
	"github.com/thelolagemann/gomeboy/internal/types"
)

// MemoryBankedCartridge2 is a cartridge that supports ROM
// sizes up to 2Mbit (16 banks of 16KiB) and includes an internal
// 512x4 bit RAM array, which is unique amongst MBC cartridges.
type MemoryBankedCartridge2 struct {
	rom    []byte
	ram    []byte
	header *Header

	ramg bool
	romb uint8
}

func (m *MemoryBankedCartridge2) Load(s *types.State) {
	s.ReadData(m.ram)
	m.ramg = s.ReadBool()
	m.romb = s.Read8()
}

func (m *MemoryBankedCartridge2) Save(s *types.State) {
	s.WriteData(m.ram)
	s.WriteBool(m.ramg)
	s.Write8(m.romb)
}

// NewMemoryBankedCartridge2 returns a new MemoryBankedCartridge2 cartridge.
func NewMemoryBankedCartridge2(rom []byte, header *Header) *MemoryBankedCartridge2 {
	header.b.Lock(0xA000)
	return &MemoryBankedCartridge2{
		rom:    rom,
		ram:    make([]byte, 512),
		header: header,
		romb:   0x01,
	}
}

func (m *MemoryBankedCartridge2) Write(address uint16, value uint8) {
	switch {
	case address <= 0x3FFF:
		if (address & 0x100) == 0x100 {
			m.romb = value & 0x0F
			if m.romb == 0 {
				m.romb = 1
			}

			// check to see if banks exceed rom
			if int(m.romb)*0x4000 >= len(m.rom) {
				m.romb = m.romb % uint8(len(m.rom)/0x4000)
			}

			// copy from bank to bus
			m.header.b.CopyTo(0x4000, 0x8000, m.rom[int(m.romb)*0x4000:])
		} else {
			if m.ramg {
				// copy data from bus to ram
				m.header.b.CopyFrom(0xA000, 0xA200, m.ram)
			}
			m.ramg = (value & 0x0F) == 0x0A

			if m.ramg {
				// only 2048 bits, so we need to account for RAM wrap around
				// TODO handle on RAM write
				for i := 0; i < 16; i++ {
					m.header.b.CopyTo(0xA000+(uint16(i)*0x200), 0xA200+(uint16(i)*0x200), m.ram)
				}
				m.header.b.Unlock(0xA000)
			} else {
				m.header.b.Lock(0xA000)
			}
		}
	case address >= 0xA000 && address <= 0xBFFF:
		// make sure to account for ram wrap around by setting the
		if m.ramg {
			for i := 0; i < 16; i++ {
				m.header.b.Set(0xA000+(uint16(i)*0x200)+address&0x01ff, value|0xF0)
			}
		}
	}

}
