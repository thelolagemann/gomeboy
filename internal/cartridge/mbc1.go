package cartridge

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
)

// MemoryBankedCartridge1 represents a MemoryBankedCartridge1 cartridge. This cartridge type has external RAM and
// supports switching between 2 ROM banks and 4 RAM banks.
type MemoryBankedCartridge1 struct {
	rom    []byte
	ram    []byte
	header *Header

	// the ramg register is used to enable access to the cartridge SRAM
	// if one exists on the cartridge. RAM access is disabled by default,
	// but can be enabled by writing 0b1010 to the lower 4 bits of the
	// ramg register, and disabled by writing any other value.
	ramg bool // 0x0000 - 0x1FFF

	// bank1 is a 5-bit value that selects the lower 5 bits of the ROM bank
	// when the CPU accesses 0x4000 - 0x7FFF. bank1 is initialized to 0x01
	// and attempting to write 0b00000 to it will write 0b00001 instead. This
	// makes it impossible to read banks 0x00, 0x20, 0x40, and 0x60 from the
	// 0x4000 - 0x7FFF range, because those bank numbers have 0b00000 in the
	// lower 5 bits. Due to the zero value adjustment, requesting any of
	// these banks will instead request the next bank in the sequence.
	bank1 uint8 // 0x2000 - 0x3FFF

	// bank2 can be used as the upper bits of the ROM bank number, os as the
	// 2-bit RAM bank number. Unlinke bank1, bank2 doesn't disallow 0, so all
	// 2-bit values are valid.
	bank2 uint8 // 0x4000 - 0x5FFF

	// mode determines how the bank2 register value is used during memory
	// accesses. If mode is 0, bank 2 affects access to 0x4000 - 0x7FFF only,
	// if mode is 1, bank2 affects access to 0x0000 - 0x7FFF and 0xA000 - 0xBFFF.
	mode bool // 0x6000 - 0x7FFF

	isMultiCart bool
}

func (m *MemoryBankedCartridge1) Load(s *types.State) {
	s.ReadData(m.ram)
	m.ramg = s.ReadBool()
	m.bank1 = s.Read8()
	m.bank2 = s.Read8()
	m.mode = s.ReadBool()
	m.isMultiCart = s.ReadBool()
}

func (m *MemoryBankedCartridge1) Save(s *types.State) {
	s.WriteData(m.ram)
	s.WriteBool(m.ramg)
	s.Write8(m.bank1)
	s.Write8(m.bank2)
	s.WriteBool(m.mode)
	s.WriteBool(m.isMultiCart)
}

// NewMemoryBankedCartridge1 returns a new MemoryBankedCartridge1 cartridge.
func NewMemoryBankedCartridge1(rom []byte, header *Header) *MemoryBankedCartridge1 {
	m := &MemoryBankedCartridge1{
		rom:    rom,
		ram:    make([]byte, header.RAMSize),
		header: header,
		bank1:  0x01,
	}
	m.checkMultiCart()
	m.header.b.Lock(io.RAM)
	return m
}

// Write attempts to switch the ROM or RAM bank.
func (m *MemoryBankedCartridge1) Write(address uint16, value uint8) {
	switch {
	case address < 0x2000:
		m.handleRAMBank(func() {
			m.ramg = value&0x0f == 0x0a
		})
	case address < 0x4000:
		// bank1 is a 5-bit value, so the upper 3 bits are ignored.
		value &= 0x1F
		if value == 0 {
			value = 1
		}
		m.bank1 = value
		if m.isMultiCart {
			m.bank1 &= 0x0F
		}

		m.handleBanking()
	case address < 0x6000:
		m.handleRAMBank(func() {
			// bank2 is a 2-bit value, so the upper 6 bits are ignored.
			m.bank2 = value & 0b11

			m.handleBanking()

			// if mode true, also account for 0x0000-0x4000 range
			if m.mode {
				bankNumber := m.bank2 << m.bankShift()
				if bankNumber >= uint8(len(m.rom)/0x4000) {
					bankNumber = bankNumber % (uint8(len(m.rom) / 0x4000))
				}

				m.header.b.CopyTo(0x0000, 0x4000, m.rom[int(bankNumber)*0x4000:])
			} else {
				m.header.b.CopyTo(0x0000, 0x4000, m.rom[0x0000:])
			}
		})
	case address < 0x8000:
		m.handleRAMBank(func() {
			m.mode = value&1 == 1
		})

	case address >= 0xA000 && address < 0xC000:
		// if there is no RAM or RAM is disabled, do nothing
		if len(m.ram) == 0 || !m.ramg {
			return
		}

		m.header.b.Set(address, value)

	default:
		panic(fmt.Sprintf("mbc1: illegal write to address: %X", address))
	}
}

func (m *MemoryBankedCartridge1) handleRAMBank(f func()) {
	// if RAM is enabled and banked, we need to copy from the bus to
	// the cartridge RAM before changing the bank
	if m.ramg && len(m.ram) > 0 {
		if !m.mode || m.header.RAMSize == 8192 {
			m.header.b.CopyFrom(0xA000, 0xC000, m.ram)
		} else if m.mode {
			offset := uint16(m.bank2&0x03) * 0x2000 // only use the lower 2 bits
			m.header.b.CopyFrom(0xA000, 0xC000, m.ram[offset:offset+0x2000])
		}
	}
	f()

	// now if RAM is enabled, we need to copy data from bank to bus
	if m.ramg && len(m.ram) > 0 {
		if !m.mode || m.header.RAMSize == 8192 {
			m.header.b.CopyTo(0xA000, 0xC000, m.ram)
		} else if m.mode {
			offset := uint16(m.bank2&0x03) * 0x2000 // only use the lower 2 bits
			m.header.b.CopyTo(0xA000, 0xC000, m.ram[offset:offset+0x2000])
		}
		m.header.b.Unlock(io.RAM)
	} else if !m.ramg {
		m.header.b.Lock(io.RAM)
	}
}

func (m *MemoryBankedCartridge1) handleBanking() {
	bankNumber := m.bank1 | m.bank2<<m.bankShift()
	if bankNumber >= uint8(len(m.rom)/0x4000) {
		bankNumber = bankNumber % (uint8(len(m.rom) / 0x4000))
	}

	// copy data from bank to bus
	m.header.b.CopyTo(0x4000, 0x8000, m.rom[int(bankNumber)*0x4000:])
}

// SaveRAM returns the RAM of the cartridge.
func (m *MemoryBankedCartridge1) SaveRAM() []byte {
	return m.ram
}

// LoadRAM loads the RAM of the cartridge.
func (m *MemoryBankedCartridge1) LoadRAM(data []byte) {
	copy(m.ram, data)
}

var logo = [48]byte{
	0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B,
	0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D,
	0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
	0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99,
	0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC,
	0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E,
}

func (m *MemoryBankedCartridge1) checkMultiCart() {
	// heuristics to detect multicart
	if m.header.ROMSize == (1024 * 1024) {
		count := 0
		compare := true

		for bank := 0; bank < 4; bank++ {
			for addr := 0x0104; addr <= 0x0133; addr++ {
				if m.rom[bank*0x40000+addr] != logo[addr-0x0104] {
					compare = false
					break
				}
			}

			if compare {
				count += 1
			}
		}
		if count > 1 {
			m.isMultiCart = true
		}
	}
}

func (m *MemoryBankedCartridge1) bankShift() uint8 {
	if m.isMultiCart {
		return 4
	}
	return 5
}
