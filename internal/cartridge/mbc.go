package cartridge

import (
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
}

func (m *memoryBankedCartridge) LoadRAM(b []byte, bu *io.Bus) {
	m.ram = b
	bu.CopyTo(0xA000, 0xC000, m.ram)
}

func (m *memoryBankedCartridge) RAM() []byte {
	return m.ram
}

func newMemoryBankedCartridge(rom []byte, ramSize uint) *memoryBankedCartridge {
	return &memoryBankedCartridge{
		rom:     rom,
		ram:     make([]byte, ramSize),
		romBank: 1,
	}
}
