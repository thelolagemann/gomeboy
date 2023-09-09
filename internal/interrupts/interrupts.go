package interrupts

import (
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
	Flag   uint8 // interrupt Flag (types.IF)
	Enable uint8 // interrupt Enable (types.IE)
}

// NewService returns a new Service.
func NewService() *Service {
	s := &Service{}
	// setup registers
	types.RegisterHardware(
		types.IF,
		func(v uint8) {
			s.Flag = v & 0x1F // only the first 5 bits are used
		}, func() uint8 {
			return s.Flag | 0xE0 // the upper 3 bits are always set
		},
	)
	types.RegisterHardware(
		types.IE,
		func(v uint8) {
			s.Enable = v
		}, func() uint8 {
			return s.Enable
		},
	)

	return s
}

// HasInterrupts returns true if there are any interrupts
// that are requested and enabled.
func (s *Service) HasInterrupts() bool {
	return s.Enable&s.Flag != 0
}

// Request requests the specified interrupt, by setting
// the corresponding bit in the Flag register.
func (s *Service) Request(flag uint8) {
	s.Flag |= flag
}

// Vector returns the currently serviced interrupt vector,
// or 0 if no interrupt is being serviced. This function
// will also clear the corresponding bit in the Flag
// register.
func (s *Service) Vector() uint16 {
	if s.Enable&s.Flag == 0 {
		return 0
	}
	for i := uint8(0); i < 5; i++ {
		// get the flag for the current interrupt
		flag := uint8(1 << i)

		// check if the interrupt is requested and enabled
		if s.Flag&(flag) != 0 && s.Enable&(flag) != 0 {
			// clear the interrupt flag and return the vector
			s.Flag = s.Flag ^ flag
			return uint16(0x0040 + i*8)
		}
	}

	return 0
}

var _ types.Stater = (*Service)(nil)

// Load implements the types.Stater interface.
//
// The values are loaded in the following order:
//   - Flag (uint8)
//   - Enable (uint8)
func (s *Service) Load(st *types.State) {
	s.Flag = st.Read8()
	s.Enable = st.Read8()
}

// Save implements the types.Stater interface.
//
// The values are saved in the following order:
//   - Flag (uint8)
//   - Enable (uint8)
func (s *Service) Save(st *types.State) {
	st.Write8(s.Flag)
	st.Write8(s.Enable)
}
