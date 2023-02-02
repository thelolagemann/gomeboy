package interrupts

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
)

// Address is an address of an interrupt. When an interrupt occurs,
// the CPU jumps to the specified interrupt address.
type Address uint16

const (
	// VBlank is the VBL interrupt address.
	VBlank Address = 0x40
	// LCD is the LCD interrupt address.
	LCD Address = 0x48
	// Timer is the Timer interrupt address.
	Timer Address = 0x50
	// Serial is the Serial interrupt address.
	Serial Address = 0x58
	// Joypad is the Joypad interrupt address.
	Joypad Address = 0x60
)

// Flag is an interrupt flag, which simply specifies what bit of the
// interrupt registers is used to access the interrupt.
type Flag uint8

const (
	// VBlankFlag is the VBL interrupt flag (bit 0).
	VBlankFlag Flag = 0x00
	// LCDFlag is the LCD interrupt flag (bit 1).
	LCDFlag Flag = 0x01
	// TimerFlag is the Timer interrupt flag (bit 2).
	TimerFlag Flag = 0x02
	// SerialFlag is the Serial interrupt flag (bit 3).
	SerialFlag Flag = 0x03
	// JoypadFlag is the Joypad interrupt flag (bit 4).
	JoypadFlag Flag = 0x04
)

// Service represents the current state of the interrupts.
type Service struct {
	// Flag is the Interrupt FlagRegister. (0xFF0F)
	Flag uint8
	// Enable is the Interrupt EnableRegister. (0xFFFF)
	Enable uint8

	// IME is the Interrupt Master Enable flag.
	IME bool
}

// NewService returns a new Service.
func NewService() *Service {
	s := &Service{
		Flag:   0,
		Enable: 0,
		IME:    false,
	}
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

// Request requests an interrupt.
func (s *Service) Request(flag Flag) {
	s.Flag = s.Flag | 1<<flag
}

// Vector returns the currently serviced interrupt vector,
// or 0 if no interrupt is being serviced. This function
// will also clear the interrupt flag.
func (s *Service) Vector() Address {
	for i := uint8(0); i < 5; i++ {
		if s.Flag&(1<<i) != 0 && s.Enable&(1<<i) != 0 {
			// clear the interrupt flag and return the vector
			s.Flag = s.Flag ^ 1<<i
			return Address(0x0040 + i*8)
		}
	}

	return 0
}
