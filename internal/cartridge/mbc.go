package cartridge

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
)

type MemoryBankController interface {
	Write(address uint16, value uint8)
	RAM() []byte
	LoadRAM([]byte, *io.Bus)
}

type memoryBankedCartridge struct {
	rom, ram []byte
	romBank  uint16
	ramBank  uint8

	ramEnabled bool

	*Header
}

func (m *memoryBankedCartridge) LoadRAM(b []byte, bu *io.Bus) {
	m.ram = b
	bu.CopyTo(0xA000, 0xC000, m.ram)
}

func (m *memoryBankedCartridge) RAM() []byte {
	return m.ram
}

// setROMBank updates the ROM bank of the cartridge and copies
// the new ROM bank to the bus.
func (m *memoryBankedCartridge) setROMBank(bank uint16, canBeZero bool) {
	m.romBank = bank

	if !canBeZero && m.romBank == 0 {
		m.romBank = 1
	}

	if int(m.romBank)*0x4000 >= len(m.rom) {
		m.romBank = uint16(int(m.romBank) % (len(m.rom) / 0x4000))
	}

	// copy from bank to bus
	m.b.CopyTo(0x4000, 0x8000, m.rom[int(m.romBank)*0x4000:])
}

func newMemoryBankedCartridge(rom []byte, h *Header) *memoryBankedCartridge {
	fmt.Println(h.RAMSize)
	return &memoryBankedCartridge{
		rom:     rom,
		ram:     make([]byte, h.RAMSize),
		romBank: 1,
		Header:  h,
	}
}
