package interrupts

import (
	"github.com/thelolagemann/gomeboy/internal/types"
)

const (
	// VBlankFlag is the VBlank interrupt flag (bit 0),
	// which is requested every time the PPU enters
	// VBlank mode (lcd.VBlank).
	VBlankFlag = types.Bit0
	// LCDFlag is the LCD interrupt flag (bit 1), which
	// is requested by the LCD STAT register (types.STAT),
	// when certain conditions are met.
	LCDFlag = types.Bit1
	// TimerFlag is the Timer interrupt flag (bit 2),
	// which is requested when the timer overflows,
	// (types.TIMA > 0xFF).
	TimerFlag = types.Bit2
	// SerialFlag is the Serial interrupt flag (bit 3),
	// which is requested when a serial transfer is
	// completed.
	SerialFlag = types.Bit3
	// JoypadFlag is the Joypad interrupt Flag (bit 4),
	// which is requested when any of types.P1 bits 0-3
	// go from high to low, if the corresponding select
	// bit (types.P1 bit 4 or 5) is set to 0.
	JoypadFlag = types.Bit4
)
