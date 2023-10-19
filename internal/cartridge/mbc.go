package cartridge

import "github.com/thelolagemann/gomeboy/internal/types"

type MemoryBankController interface {
	Write(address uint16, value uint8)

	types.Stater
}
