package interrupts

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
)

const (
	// VBlankFlag is the VBlank interrupt flag (bit 0),
	// which is requested every time the PPU enters
	// VBlank mode (lcd.VBlank).
	VBlankFlag = types.Bit0
	// LCDFlag is the LCD interrupt flag (bit 1), which
	// is requested by the LCD STAT register (types.STAT),
	// when certain conditions are met.
	LCDFlag = types.Bit1
	// TimerFlag is the Timer interrupt flag (bit 2),
	// which is requested when the timer overflows,
	// (types.TIMA > 0xFF).
	TimerFlag = types.Bit2
	// SerialFlag is the Serial interrupt flag (bit 3),
	// which is requested when a serial transfer is
	// completed.
	SerialFlag = types.Bit3
	// JoypadFlag is the Joypad interrupt Flag (bit 4),
	// which is requested when any of types.P1 bits 0-3
	// go from high to low, if the corresponding select
	// bit (types.P1 bit 4 or 5) is set to 0.
	JoypadFlag = types.Bit4
)

// Service is the interrupt service, used to request
// interrupts and to get the current interrupt vector.
//
// When an interrupt is requested, the corresponding bit
// in the Flag register is set. When an interrupt is
// enabled, the corresponding bit in the Enable register
// is set. When an interrupt is requested and enabled,
// and the IME is set, the CPU will jump to the interrupt
// vector, and the corresponding bit in the Flag register
// will be cleared.
//
// The IME is set by the DI, EI and RETI instructions,
// and it is used to disable and enable interrupts.
type Service struct {
	b *io.Bus
}

// NewService returns a new Service.
func NewService(b *io.Bus) *Service {
	s := &Service{
		b: b,
	}
	b.ReserveAddress(types.IF, func(b byte) byte {
		return b | 0xE0 // the upper 3 bits are always set
	})
	b.ReserveAddress(types.IE, func(b byte) byte {
		return b | 0xE0 // the upper 3 bits are always set
	})

	return s
}

// HasInterrupts returns true if there are any interrupts
// that are requested and enabled.
func (s *Service) HasInterrupts() bool {
	return s.b.Get(types.IF)&s.b.Get(types.IE) != 0
}

// Request requests the specified interrupt, by setting
// the corresponding bit in the Flag register.
func (s *Service) Request(flag uint8) {
	s.b.SetBit(types.IF, flag)
}

// Vector returns the currently serviced interrupt vector,
// or 0 if no interrupt is being serviced. This function
// will also clear the corresponding bit in the Flag
// register.
func (s *Service) Vector(from uint8) uint16 {
	for i := uint8(0); i < 5; i++ {
		// get the flag for the current interrupt
		flag := uint8(1 << i)

		// check if the interrupt is requested and enabled
		if from&s.b.Get(types.IF)&flag == flag {
			// clear the flag
			s.b.ClearBit(types.IF, flag)

			// return vector
			return uint16(0x0040 + i*8)
		}
	}

	return 0
}
