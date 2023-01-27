// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/internal/types/registers"
)

// Controller is a timer controller. It is used to generate
// interrupts at a specific frequency. The frequency can be
// configured using the registers.TAC register.
type Controller struct {
	internalDivider uint16

	div  *registers.Hardware
	tima *registers.Hardware
	tma  *registers.Hardware
	tac  *registers.Hardware

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
	c.div = registers.NewHardware(
		registers.DIV,
		registers.WithReadFunc(func(h *registers.Hardware, address uint16) uint8 {
			return uint8(c.internalDivider >> 8)
		}),
		registers.WithWriteFunc(func(h *registers.Hardware, address uint16, value uint8) {
			c.internalDivider = 0
		}),
	)

	c.tima = registers.NewHardware(
		registers.TIMA,
		registers.IsReadable(),
		registers.WithWriteFunc(func(h *registers.Hardware, address uint16, value uint8) {
			// writes to TIMA are ignored if written the same tick it is
			// reloading
			if c.ticksSinceOverflow != 5 {
				h.Set(value)
				c.overflow = false
				c.ticksSinceOverflow = 0
			}
		}),
	)

	c.tma = registers.NewHardware(
		registers.TMA,
		registers.IsReadable(),
		registers.WithWriteFunc(func(h *registers.Hardware, address uint16, value uint8) {
			h.Set(value)
			// if you write to TMA the same tick that TIMA is reloading,
			// TIMA will be set to the new value of TMA
			if c.ticksSinceOverflow == 5 {
				c.tima.Set(value)
			}
		}),
	)

	c.tac = registers.NewHardware(
		registers.TAC,
		registers.IsReadableMasked(types.CombineMasks(types.Mask0, types.Mask1, types.Mask2)),
		registers.WithWriteFunc(func(h *registers.Hardware, address uint16, value uint8) {
			wasEnabled := c.enabled
			oldBit := c.currentBit

			h.Set(value & 0b111)
			c.currentBit = 1 << bits[value&0b11]
			c.enabled = (value & 0x4) == 0x4

			c.timaGlitch(wasEnabled, oldBit)
		}),
	)
}

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		internalDivider: 0,

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
	c.internalDivider++

	bit := (c.internalDivider&c.currentBit) != 0 && c.enabled

	// detect a falling edge
	if c.lastBit && !bit {
		// increment timer
		c.tima.Increment()

		// check for overflow
		if c.tima.Read() == 0 {
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
			c.tima.Set(c.tma.Value())
		} else if c.ticksSinceOverflow == 6 {
			c.overflow = false
			c.ticksSinceOverflow = 0
		}
	}
}

func (c *Controller) multiplexer() uint16 {
	switch c.tac.Read() & 0x03 {
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

	if c.internalDivider&oldBit != 0 {
		if !c.enabled || !(c.internalDivider&c.currentBit != 0) {
			c.tima.Increment()

			if c.tima.Value() == 0 {
				c.tima.Set(c.tma.Value())
				c.irq.Request(interrupts.TimerFlag)
			}

			c.lastBit = false
		}
	}
}

var bits = [4]uint8{9, 3, 5, 7}
