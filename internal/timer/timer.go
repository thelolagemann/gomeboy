// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
)

// Controller is a timer controller. It is used to generate
// interrupts at a specific frequency. The frequency can be
// configured using the registers.TAC register.
type Controller struct {
	divider uint16

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

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service) *Controller {
	return &Controller{
		divider: 0,
		tima:    0,
		tma:     0,
		tac:     0,

		irq:        irq,
		currentBit: 1 << bits[0],
	}
}

// HasDoubleSpeed returns true as the timer controller responds to
// double speed mode.
func (c *Controller) HasDoubleSpeed() bool {
	return true
}

// Tick ticks the timer controller.
func (c *Controller) Tick() {
	// increment divider register
	c.divider++

	bit := (c.divider&c.currentBit) != 0 && c.enabled

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

// Read reads a byte from the timer controller.
func (c *Controller) Read(addr uint16) uint8 {
	switch addr {
	case registers.DIV:
		return uint8(c.divider >> 8)
	case registers.TIMA:
		return c.tima
	case registers.TMA:
		return c.tma
	case registers.TAC:
		return c.tac | 0xF8
	}
	panic(fmt.Sprintf("timer: illegal read from %x", addr))
}

// Write writes a byte to the timer controller.
func (c *Controller) Write(addr uint16, val uint8) {
	switch addr {
	case registers.DIV:
		c.divider = 0
	case registers.TIMA:
		// writes to TIMA are ignored if written the same tick it is
		// reloading
		if c.ticksSinceOverflow != 5 {
			c.tima = val
			c.overflow = false
			c.ticksSinceOverflow = 0
		}
	case registers.TMA:
		c.tma = val
		// if you write to TMA the same tick that TIMA is reloading,
		// TIMA will be set to the new value of TMA
		if c.ticksSinceOverflow == 5 {
			c.tima = val
		}
	case registers.TAC:
		wasEnabled := c.enabled
		oldBit := c.currentBit

		c.tac = val & 0b111
		c.currentBit = 1 << bits[c.tac&0b11]
		c.enabled = (c.tac & 0x4) == 0x4

		c.timaGlitch(wasEnabled, oldBit)
	default:
		panic(fmt.Sprintf("timer: illegal write to %x", addr))
	}
}

// timaGlitch handles the glitch that occurs when the timer is enabled
// or disabled.
func (c *Controller) timaGlitch(wasEnabled bool, oldBit uint16) {
	if !wasEnabled {
		return
	}

	if c.divider&oldBit != 0 {
		if !c.enabled || !(c.divider&c.currentBit != 0) {
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
