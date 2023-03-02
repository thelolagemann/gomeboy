package cartridge

import "github.com/thelolagemann/go-gameboy/internal/types"

type MemoryBankController interface {
	Read(address uint16) uint8
	Write(address uint16, value uint8)

	types.Stater
}
