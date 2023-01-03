package io

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/pkg/bits"
)

type InterruptAddress = uint8

const (
	// InterruptVBL is the VBL interrupt address.
	InterruptVBL InterruptAddress = 0x0040
	// InterruptLCD is the LCD interrupt address.
	InterruptLCD InterruptAddress = 0x0048
	// InterruptTimer is the Timer interrupt address.
	InterruptTimer InterruptAddress = 0x0050
	// InterruptSerial is the Serial interrupt address.
	InterruptSerial InterruptAddress = 0x0058
	// InterruptJoypad is the Joypad interrupt address.
	InterruptJoypad InterruptAddress = 0x0060
)

// InterruptFlag is an interrupt flag.
type InterruptFlag uint8

const (
	InterruptVBLFlag    InterruptFlag = 0x00
	InterruptLCDFlag    InterruptFlag = 0x01
	InterruptTimerFlag  InterruptFlag = 0x02
	InterruptSerialFlag InterruptFlag = 0x03
	InterruptJoypadFlag InterruptFlag = 0x04
)

const (
	// InterruptFlagRegister is the register for the interrupt flags. (R/W)
	//
	//  Bit 0: V-Blank  Interrupt Request (INT 40h)  (1=Request)
	//  Bit 1: LCD STAT Interrupt Request (INT 48h)  (1=Request)
	//  Bit 2: Timer    Interrupt Request (INT 50h)  (1=Request)
	//  Bit 3: Serial   Interrupt Request (INT 58h)  (1=Request)
	//  Bit 4: Joypad   Interrupt Request (INT 60h)  (1=Request)
	InterruptFlagRegister uint16 = 0xFF0F
	// InterruptEnableRegister is the register for the interrupt enable flags. (R/W)
	InterruptEnableRegister uint16 = 0xFFFF
)

// Interrupts represents the InterruptAddress registers, IF and IE, as well
// as the IME (Interrupt Master Enable) flag.
type Interrupts struct {
	// IF is the Interrupt Flag register.
	IF uint8
	// IE is the Interrupt Enable register.
	IE uint8
	// IME is the Interrupt Master Enable flag.
	IME bool
}

// NewInterrupts returns a new Interrupts.
func NewInterrupts() *Interrupts {
	return &Interrupts{
		IF:  0,
		IE:  0,
		IME: false,
	}
}

// Request requests an interrupt.
func (i *Interrupts) Request(flag InterruptFlag) {
	i.IF = bits.Set(i.IF, uint8(flag))
}

// Read reads from the Interrupts.
func (i *Interrupts) Read(addr uint16) uint8 {
	fmt.Println(fmt.Sprintf("read from interrupt register: %04x", addr))
	switch addr {
	case InterruptFlagRegister:
		return i.IF | 0xE0
	case InterruptEnableRegister:
		return i.IE
	}
	panic(fmt.Sprint("illegal read from interrupt register: ", addr))
}

// Write writes to the Interrupts.
func (i *Interrupts) Write(addr uint16, value uint8) {
	fmt.Println(fmt.Sprintf("write to interrupt register: %04x, %08b", addr, value))
	switch addr {
	case InterruptFlagRegister:
		i.IF = value
	case InterruptEnableRegister:
		i.IE = value
	default:
		panic(fmt.Sprint("illegal write to interrupt register: ", addr))
	}
}
