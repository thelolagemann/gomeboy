// Package joypad provides an emulation of the Game Boy
// joypad. It is responsible for reading back the state of the
// joypad, as well as handling the interrupt when a button is
// pressed.
package joypad

import (
	"github.com/thelolagemann/go-gameboy/pkg/bits"
)

// Button represents a physical button on the Game Boy.
type Button = uint8

const (
	// ButtonA is the A button.
	ButtonA Button = 0x01
	// ButtonB is the B button.
	ButtonB = 0x02
	// ButtonSelect is the Select button.
	ButtonSelect = 0x04
	// ButtonStart is the Start button.
	ButtonStart = 0x08
	// ButtonRight is the Right button.
	ButtonRight = 0x10
	// ButtonLeft is the Left button.
	ButtonLeft = 0x20
	// ButtonUp is the Up button.
	ButtonUp = 0x40
	// ButtonDown is the Down button.
	ButtonDown = 0x80
)

// State represents the state of the joypad.
type State struct {
	// Register is the joypad register.
	Register byte
	// State is the current state of the joypad.
	State Button
}

// New returns a new joypad state.
func New() *State {
	return &State{
		Register: 0x3F,
	}
}

// Read returns the current state of the joypad.
func (s *State) Read() uint8 {
	if s.Register&0x10 == 0 {
		return s.Register & ^(s.State >> 4)
	}
	if s.Register&0x20 == 0 {
		return s.Register & ^(s.State & 0x0F)
	}

	return s.Register | 0x0F
}

// Write writes the value to the joypad.
func (s *State) Write(value byte) {
	s.Register = (s.Register & 0xCF) | (value & 0x30)
}

// Press presses the given key on the joypad, and returns
// whether an interrupt should be triggered.
func (s *State) Press(key Button) bool {
	prevUnset := false
	if bits.Test(s.State, key) {
		prevUnset = true
	}
	reqInt := false
	s.State |= key

	// only trigger interrupt if the button was previously unset
	// and the game is listening for it
	if key <= ButtonStart && !bits.Test(s.Register, 5) {
		reqInt = true
	} else if key > ButtonStart && !bits.Test(s.Register, 4) {
		reqInt = true
	}

	if !prevUnset && reqInt {
		return true
	}

	return false
}

// Release releases the given key.
func (s *State) Release(key Button) {
	s.State &^= key
}

type Inputs struct {
	Pressed, Released []Button
}

// ProcessInputs processes the inputs.
func (s *State) ProcessInputs(inputs Inputs) bool {
	interrupt := false
	for _, key := range inputs.Pressed {
		if reqInt := s.Press(key); reqInt {
			interrupt = true
		}
	}
	for _, key := range inputs.Released {
		s.Release(key)
	}

	return interrupt
}
