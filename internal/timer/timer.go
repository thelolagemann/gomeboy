// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

// Controller is a timer controller. It is used to generate
// interrupts at a specific frequency. The frequency can be
// configured using the types.TAC register.
type Controller struct {
	Div                uint16
	currentBit         uint16
	tima               uint8
	ticksSinceOverflow uint8
	tma                uint8
	tac                uint8

	enabled  bool
	lastBit  bool
	overflow bool

	irq *interrupts.Service
}

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq:        irq,
		currentBit: bits[0],
		Div:        0xABCC,
		tac:        0xF8,
	}
	// set up types
	types.RegisterHardware(
		types.DIV,
		func(v uint8) {
			c.Div = 0 // any write to DIV resets it
		},
		func() uint8 {
			// return bits 6-13 of divider register
			return uint8(c.Div >> 8) // TODO actually return bits 6-13
		},
	)
	types.RegisterHardware(
		types.TIMA,
		func(v uint8) {
			// writes to TIMA are ignored if written the same tick it is
			// reloading
			if c.ticksSinceOverflow != 5 {
				c.tima = v
				c.overflow = false
				c.ticksSinceOverflow = 0
			}
		}, func() uint8 {
			return c.tima
		},
	)
	types.RegisterHardware(
		types.TMA,
		func(v uint8) {
			c.tma = v
			// if you write to TMA the same tick that TIMA is reloading,
			// TIMA will be set to the new value of TMA
			if c.ticksSinceOverflow == 5 {
				c.tima = v
			}
		}, func() uint8 {
			return c.tma
		},
	)
	types.RegisterHardware(
		types.TAC,
		func(v uint8) {
			wasEnabled := c.enabled
			oldBit := c.currentBit
			// 00 = shift by 9 bits
			// 01 = shift by 3 bits
			// 10 = shift by 5 bits
			// 11 = shift by 7 bits

			c.tac = v
			c.currentBit = bits[v&0b11]
			c.enabled = (v & 0x4) == 0x4

			c.timaGlitch(wasEnabled, oldBit)
		}, func() uint8 {
			return c.tac | 0b11111000
		},
	)

	return c
}

// Tick ticks the timer controller.
func (c *Controller) Tick() {
	// increment internalDivider register
	c.Div++

	// check if timer is enabled
	if !c.enabled {
		return
	}

	newBit := (c.Div & c.currentBit) != 0

	// detect a falling edge
	if c.lastBit && !newBit {
		// increment timer
		c.tima++

		// check for overflow
		if c.tima == 0 {
			c.overflow = true
			c.ticksSinceOverflow = 0
		}
	}

	// update last bit
	c.lastBit = newBit

	// check for overflow
	if c.overflow {
		c.ticksSinceOverflow++

		// handle ticks since overflow
		if c.ticksSinceOverflow == 4 {
			c.irq.Request(interrupts.TimerFlag)
		} else if c.ticksSinceOverflow == 5 {
			c.tima = c.tma
		} else if c.ticksSinceOverflow == 6 {
			c.overflow = false
			c.ticksSinceOverflow = 0
		}
	}
}

// timaGlitch handles the glitch that occurs when the timer is enabled
// or disabled.
func (c *Controller) timaGlitch(wasEnabled bool, oldBit uint16) {
	if !wasEnabled {
		return
	}

	if c.Div&oldBit != 0 {
		if !c.enabled || !(c.Div&c.currentBit != 0) {
			c.tima++

			if c.tima == 0 {
				c.tima = c.tma
				c.irq.Request(interrupts.TimerFlag)
			}

			c.lastBit = false
		}
	}
}

var bits = [4]uint16{512, 8, 32, 128}
