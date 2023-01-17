package interrupts

import "fmt"

// Address is an address of an interrupt.
type Address = uint16

const (
	// VBlank is the VBL interrupt address.
	VBlank Address = 0x0040
	// LCD is the LCD interrupt address.
	LCD Address = 0x0048
	// Timer is the Timer interrupt address.
	Timer Address = 0x0050
	// Serial is the Serial interrupt address.
	Serial Address = 0x0058
	// Joypad is the Joypad interrupt address.
	Joypad Address = 0x0060
)

// Flag is an interrupt flag.
type Flag = uint8

const (
	VBlankFlag Flag = 0x00
	LCDFlag    Flag = 0x01
	TimerFlag  Flag = 0x02
	SerialFlag Flag = 0x03
	JoypadFlag Flag = 0x04
)

const (
	// FlagRegister is the register for the interrupt flags. (R/W)
	//
	//  Bit 0: V-Blank  Interrupt Request (INT 40h)  (1=Request)
	//  Bit 1: LCD STAT Interrupt Request (INT 48h)  (1=Request)
	//  Bit 2: Timer    Interrupt Request (INT 50h)  (1=Request)
	//  Bit 3: Serial   Interrupt Request (INT 58h)  (1=Request)
	//  Bit 4: Joypad   Interrupt Request (INT 60h)  (1=Request)
	FlagRegister uint16 = 0xFF0F
	// EnableRegister is the register for the interrupt enable flags. (R/W)
	EnableRegister uint16 = 0xFFFF
)

// Service represents the current state of the interrupts.
type Service struct {
	// Flag is the Interrupt FlagRegister. (0xFF0F)
	Flag uint8
	// Enable is the Interrupt EnableRegister. (0xFFFF)
	Enable uint8

	// IME is the Interrupt Master Enable flag.
	IME bool

	// Enabling represents whether the IME is being enabled.
	// This is used to delay the enabling of the IME by one cycle.
	Enabling bool
}

// NewService returns a new Service.
func NewService() *Service {
	return &Service{
		Flag:     0,
		Enable:   0,
		IME:      false,
		Enabling: false,
	}
}

// Request requests an interrupt.
func (s *Service) Request(flag Flag) {
	s.Flag |= 1 << flag
}

// Clear clears the interrupt flag at the given address.
func (s *Service) Clear(flag Flag) {
	s.Flag &^= 1 << flag
}

// Read returns the value of the register at the given address.
func (s *Service) Read(address uint16) uint8 {
	switch address {
	case FlagRegister:
		return s.Flag&0b00011111 | 0b11100000
	case EnableRegister:
		return s.Enable
	}
	panic(fmt.Sprintf("interrupts\tillegal read from address %04X", address))
}

// Write writes the given value to the register at the given address.
func (s *Service) Write(address uint16, value uint8) {
	switch address {
	case FlagRegister:
		s.Flag = value
	case EnableRegister:
		s.Enable = value
	default:
		panic(fmt.Sprintf("interrupts\tillegal write to address %04X", address))
	}
}
