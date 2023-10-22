package io

import (
	"github.com/thelolagemann/gomeboy/internal/types"
)

const (
	// VBlankINT is the VBlank interrupt flag (bit 0),
	// which is requested every time the PPU enters
	// VBlank mode (lcd.VBlank).
	VBlankINT = types.Bit0
	// LCDINT is the LCD interrupt flag (bit 1), which
	// is requested by the LCD STAT register (types.STAT),
	// when certain conditions are met.
	LCDINT = types.Bit1
	// TimerINT is the Timer interrupt flag (bit 2),
	// which is requested when the timer overflows,
	// (types.TIMA > 0xFF).
	TimerINT = types.Bit2
	// SerialINT is the Serial interrupt flag (bit 3),
	// which is requested when a serial transfer is
	// completed.
	SerialINT = types.Bit3
	// JoypadINT is the Joypad interrupt Flag (bit 4),
	// which is requested when any of types.P1 bits 0-3
	// go from high to low, if the corresponding select
	// bit (types.P1 bit 4 or 5) is set to 0.
	JoypadINT = types.Bit4
)

// IRQVector returns the current interrupt vector ready to
// be serviced, and clears the interrupt from the types.IF
// register.
//
// Only one interrupt is serviced at a time, and they are
// serviced in the order of priority:
//
//   - VBlank
//   - LCD
//   - Timer
//   - Serial
//   - Joypad
//
// When an interrupt occurs, there is a chance for the interrupt
// vector to change during the execution of the dispatch handler.
// This is because the cycle at which the CPU enters the dispatch
// handler is not the same as the cycle at which the interrupt
// vector is determined.
// https://mgba.io/2018/03/09/holy-grail-bugs-revisited/
func (b *Bus) IRQVector(irq byte) uint16 {
	for i := uint8(0); i < 5; i++ {
		// get the flag for the current interrupt
		flag := uint8(1 << i)

		// check if the interrupt is requested and enabled
		if irq&b.data[types.IF]&flag == flag {
			// clear the flag
			b.data[types.IF] &= ^flag

			// return vector
			return uint16(0x0040 + i*8)
		}
	}

	// if no interrupts
	return 0
}

// RaiseInterrupt raises the specified interrupt by setting
// the flag in the types.IF register.
func (b *Bus) RaiseInterrupt(interrupt byte) {
	b.data[types.IF] |= interrupt

	// if IME is enabled, then we need to notify the CPU
	if b.CanInterrupt() {
		b.Write(0xFF7E, 0)
	}
}

// HasInterrupts returns true if there are pending interrupts.
func (b *Bus) HasInterrupts() bool {
	return b.data[types.IE]&b.data[types.IF]&0x1F != 0
}

func (b *Bus) CanInterrupt() bool {
	return b.ime && b.HasInterrupts()
}

func (b *Bus) EnableInterrupts() {
	b.ime = true

	// determine if we should tell the CPU to interrupt
	if b.HasInterrupts() {
		b.Write(0xff7e, 0)
	}
}

func (b *Bus) DisableInterrupts() {
	b.ime = false
}

func (b *Bus) InterruptsEnabled() bool {
	return b.ime
}
