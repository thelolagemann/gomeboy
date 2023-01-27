package interrupts

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
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
	Flag *registers.Hardware
	// Enable is the Interrupt EnableRegister. (0xFFFF)
	Enable *registers.Hardware

	// IME is the Interrupt Master Enable flag.
	IME bool
}

// NewService returns a new Service.
func NewService() *Service {
	return &Service{
		Flag: registers.NewHardware(
			registers.IF,
			registers.Mask(types.CombineMasks(types.Mask0, types.Mask1, types.Mask2, types.Mask3, types.Mask4)),
		),
		Enable: registers.NewHardware(
			registers.IE,
			registers.IsReadableWritable(),
		),
		IME: false,
	}
}

// Request requests an interrupt.
func (s *Service) Request(flag Flag) {
	s.Flag.Write(s.Flag.Read() | 1<<flag)
}

// Vector returns the currently serviced interrupt vector,
// or 0 if no interrupt is being serviced. This function
// will also clear the interrupt flag.
func (s *Service) Vector() Address {
	for i := uint8(0); i < 5; i++ {
		if s.Flag.Read()&(1<<i) != 0 && s.Enable.Read()&(1<<i) != 0 {
			// clear the interrupt flag and return the vector
			s.Flag.Write(s.Flag.Read() ^ 1<<i)
			return Address(0x0040 + i*8)
		}
	}

	return 0
}
