// Package joypad provides an implementation of the Game Boy
// joypad. The joypad is used to read the state of the buttons
// and the direction keys.
package joypad

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
)

const (
	// JoypadRegister is the address of the joypad register.
	JoypadRegister = 0xFF00
)

// Button represents a physical button on the Game Boy.
type Button = uint8

const (
	// ButtonA is the A button.
	ButtonA Button = iota
	// ButtonB is the B button.
	ButtonB
	// ButtonSelect is the Select button.
	ButtonSelect
	// ButtonStart is the Start button.
	ButtonStart
	// ButtonRight is the Right button.
	ButtonRight
	// ButtonLeft is the Left button.
	ButtonLeft
	// ButtonUp is the Up button.
	ButtonUp
	// ButtonDown is the Down button.
	ButtonDown
)

// State represents the state of the joypad. Select either
// action or direction buttons by writing to the register,
// and then read out bits 0-3 to get the state of the buttons.
//
//	Bit 7 - Not used
//	Bit 6 - Not used
//	Bit 5 - P15 Select Button Keys      (0=Select)
//	Bit 4 - P14 Select Direction Keys   (0=Select)
//	Bit 3 - P13 Input Down  or Start    (0=Pressed) (Read Only)
//	Bit 2 - P12 Input Up    or Select   (0=Pressed) (Read Only)
//	Bit 1 - P11 Input Left  or Button B (0=Pressed) (Read Only)
//	Bit 0 - P10 Input Right or Button A (0=Pressed) (Read Only)
type State struct {
	// Register is the joypad register. It is used to select
	// either the action or direction buttons.
	Register byte
	// State is the current state of the joypad. It is used to
	// hold the state of the buttons, the lower 4 bits are
	// used for the action buttons, and the upper 4 bits are
	// used for the direction buttons. A 0 in a bit indicates
	// that the button is pressed.
	State Button
	irq   *interrupts.Service
}

// New returns a new joypad state.
func New(irq *interrupts.Service) *State {
	return &State{
		Register: 0b1100_1111,
		State:    0b1111_1111,
		irq:      irq,
	}
}

// Read reads the joypad state.
func (s *State) Read(address uint16) uint8 {
	if address != JoypadRegister {
		panic(fmt.Sprintf("joypad: illegal read address 0x%04X", address))
	}
	// P14 and P15 are set to 1 by default, so if they are set to 0,
	// we are reading the state of the buttons.
	if !utils.Test(s.Register, 4) {
		// direction buttons are in the upper 4 bits
		return s.Register | s.State>>4
	} else if !utils.Test(s.Register, 5) {
		// action buttons are in the lower 4 bits
		return s.Register | s.State&0b0000_1111
	}

	return s.Register
}

// Write writes to the joypad register. Only bits 4 and 5 are
// writable. The rest of the bits are read-only. Bits 6 and 7
// are always set to 1.
func (s *State) Write(address uint16, value byte) {
	// handle illegal writes
	if address != JoypadRegister {
		panic(fmt.Sprintf("joypad: illegal write address 0x%04X", address))
	}
	// write bits 4 and 5 to the register
	s.Register = 0b1100_0000 | (value & 0b0011_0000)
}

// Press presses a button.
func (s *State) Press(button Button) {
	// reset the button bit in the state (0 = pressed)
	s.State = utils.Reset(s.State, button)
	s.irq.Request(interrupts.JoypadFlag)
}

// Release releases a button.
func (s *State) Release(button Button) {
	// set the button bit in the state (1 = released)
	s.State = utils.Set(s.State, button)
}
