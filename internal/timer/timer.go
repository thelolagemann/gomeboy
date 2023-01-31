// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
)

// Controller is a timer controller. It is used to generate
// interrupts at a specific frequency. The frequency can be
// configured using the registers.TAC register.
type Controller struct {
	div  uint16
	tima uint8
	tma  uint8
	tac  uint8

	enabled    bool
	currentBit uint16
	lastBit    bool

	overflow           bool
	ticksSinceOverflow uint8

	irq *interrupts.Service
}

// init initializes the timer controller, it should be called
// before the controller is returned to the caller.
func (c *Controller) init() {
	// set up registers
	registers.RegisterHardware(
		registers.DIV,
		func(v uint8) {
			c.div = 0 // any write to DIV resets it
		},
		func() uint8 {
			// return bits 6-13 of divider register
			return uint8(c.div >> 8) // TODO actually return bits 6-13
		},
	)
	registers.RegisterHardware(
		registers.TIMA,
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
	registers.RegisterHardware(
		registers.TMA,
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
	registers.RegisterHardware(
		registers.TAC,
		func(v uint8) {
			wasEnabled := c.enabled
			oldBit := c.currentBit

			c.tac = v & 0b111
			c.currentBit = 1 << bits[v&0b11]
			c.enabled = (v & 0x4) == 0x4

			c.timaGlitch(wasEnabled, oldBit)
		}, func() uint8 {
			return c.tac | 0b11111000 // bits 3-7 are always 1
		},
	)
}

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq:        irq,
		currentBit: 1 << bits[0],
	}
	c.init()

	return c
}

// HasDoubleSpeed returns true as the timer controller responds to
// double speed mode.
func (c *Controller) HasDoubleSpeed() bool {
	return true
}

// Tick ticks the timer controller.
func (c *Controller) Tick() {
	// increment internalDivider register
	c.div++

	bit := (c.div&c.currentBit) != 0 && c.enabled

	// detect a falling edge
	if c.lastBit && !bit {
		// increment timer
		c.tima++

		// check for overflow
		if c.tima == 0 {
			c.overflow = true
			c.ticksSinceOverflow = 0
		}
	}

	// update last bit
	c.lastBit = bit

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

func (c *Controller) multiplexer() uint16 {
	switch c.tac & 0x03 {
	case 0:
		return 1024
	case 1:
		return 16
	case 2:
		return 64
	case 3:
		return 256
	}

	panic("invalid multiplexer value")
}

// timaGlitch handles the glitch that occurs when the timer is enabled
// or disabled.
func (c *Controller) timaGlitch(wasEnabled bool, oldBit uint16) {
	if !wasEnabled {
		return
	}

	if c.div&oldBit != 0 {
		if !c.enabled || !(c.div&c.currentBit != 0) {
			c.tima++

			if c.tima == 0 {
				c.tima = c.tma
				c.irq.Request(interrupts.TimerFlag)
			}

			c.lastBit = false
		}
	}
}

var bits = [4]uint8{9, 3, 5, 7}
