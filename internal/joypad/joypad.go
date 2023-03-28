// Package joypad provides an implementation of the Game Boy
// joypad. The joypad is used to read the state of the buttons
// and the direction keys.
package joypad

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
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
	Register uint8
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
	s := &State{
		irq:   irq,
		State: 0xFF,
	}
	// set up the register
	types.RegisterHardware(
		types.P1,
		s.Set,
		s.Get,
		types.WithSet(func(v interface{}) {
			s.Register = v.(uint8)
		}),
	)
	return s
}

func (s *State) Get() uint8 {
	if !utils.Test(s.Register, 4) {
		return (s.Register | s.State>>4) | 0xC0
	} else if !utils.Test(s.Register, 5) {
		return (s.Register | s.State&0b0000_1111) | 0xC0
	}
	return s.Register
}

func (s *State) Set(v uint8) {
	s.Register = v | 0b1100_0000
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

var _ types.Stater = (*State)(nil)

func (s *State) Load(st *types.State) {
	s.Register = st.Read8()
	s.State = st.Read8()
}

func (s *State) Save(st *types.State) {
	st.Write8(s.Register)
	st.Write8(s.State)
}
