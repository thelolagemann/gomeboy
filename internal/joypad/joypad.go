// Package joypad provides an implementation of the Game Boy
// joypad. The joypad is used to read the state of the buttons
// and the direction keys.
package joypad

import (
	"github.com/thelolagemann/gomeboy/internal/interrupts"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/utils"
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
	// State is the current state of the joypad. It is used to
	// hold the state of the buttons, the lower 4 bits are
	// used for the action buttons, and the upper 4 bits are
	// used for the direction buttons. A 0 in a bit indicates
	// that the button is pressed.
	State Button
	b     *io.Bus
}

// New returns a new joypad state.
func New(b *io.Bus) *State {
	s := &State{
		b: b,
	}
	b.ReserveAddress(types.P1, func(v byte) byte {
		d := uint8(0xC0)
		if v&types.Bit4 == 0 {
			d |= s.State >> 4 & 0xf
		}
		if v&types.Bit5 == 0 {
			d |= s.State & 0xf
		}

		d ^= 0xf
		return d
	})
	b.ReserveSetAddress(types.P1, func(v any) {
		b.Set(types.P1, v.(byte))
	})

	return s
}

// Press presses a button.
func (s *State) Press(button Button) {
	// reset the button bit in the state (0 = pressed)
	s.State = utils.Set(s.State, types.Bit0<<button)
	s.b.SetBit(types.IF, interrupts.JoypadFlag)
}

// Release releases a button.
func (s *State) Release(button Button) {
	// set the button bit in the state (1 = released)
	s.State = utils.Reset(s.State, types.Bit0<<button)
}

var _ types.Stater = (*State)(nil)

func (s *State) Load(st *types.State) {
	s.State = st.Read8()
}

func (s *State) Save(st *types.State) {
	st.Write8(s.State)
}
