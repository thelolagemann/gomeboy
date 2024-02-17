package io

import (
	"github.com/thelolagemann/gomeboy/internal/types"
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

// Press presses a button.
func (b *Bus) Press(button Button) {
	// reset the button bit in the state (0 = pressed)
	b.buttonState |= types.Bit0 << button
	b.RaiseInterrupt(JoypadINT)
}

// Release releases a button.
func (b *Bus) Release(button Button) {
	// set the button bit in the state (1 = released)
	b.buttonState &^= types.Bit0 << button
}
