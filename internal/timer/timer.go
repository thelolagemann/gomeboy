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
	currentBit  uint16
	internalDiv uint16

	tima               uint8
	ticksSinceOverflow uint8
	tma                uint8
	tac                uint8

	Enabled   bool
	lastBit   bool
	overflow  bool
	cycleFunc func()

	irq *interrupts.Service
}

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq:        irq,
		currentBit: bits[0],
		tac:        0xF8,
	}
	// set up registers
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
			wasEnabled := c.Enabled
			oldBit := c.currentBit
			// 00 = shift by 9 bits
			// 01 = shift by 3 bits
			// 10 = shift by 5 bits
			// 11 = shift by 7 bits

			c.tac = v
			c.currentBit = bits[v&0b11]
			c.Enabled = (v & 0x4) == 0x4

			c.timaGlitch(wasEnabled, oldBit)
			c.cycleFunc()
		}, func() uint8 {
			return c.tac | 0b11111000
		},
	)

	return c
}

// TickM ticks the timer controller by 1 M-Cycle (4 T-Cycles).
func (c *Controller) TickM(sysClock uint16) {
	selectedBit := c.currentBit
	for i := 0; i < 4; i++ {
		// increment divider register
		sysClock++

		// get the new bit
		newBit := (sysClock & selectedBit) != 0

		// detect a falling edge
		if !newBit && c.lastBit {
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
			switch c.ticksSinceOverflow {
			case 4:
				c.irq.Request(interrupts.TimerFlag)
			case 5:
				c.tima = c.tma
			case 6:
				c.overflow = false
				c.ticksSinceOverflow = 0
			}
		}
	}

	c.internalDiv = sysClock
}

// timaGlitch handles the glitch that occurs when the timer is Enabled
// or disabled.
func (c *Controller) timaGlitch(wasEnabled bool, oldBit uint16) {
	if !wasEnabled {
		return
	}

	if c.internalDiv&oldBit != 0 {
		if !c.Enabled || !(c.internalDiv&c.currentBit != 0) {
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

var _ types.Stater = (*Controller)(nil)

// Load loads the state of the controller.
func (c *Controller) Load(s *types.State) {
	c.tima = s.Read8()
	c.tma = s.Read8()
	c.tac = s.Read8()

	c.Enabled = s.ReadBool()
	c.currentBit = s.Read16()
	c.lastBit = s.ReadBool()
	c.overflow = s.ReadBool()
	c.ticksSinceOverflow = s.Read8()
}

// Save saves the state of the controller.
func (c *Controller) Save(s *types.State) {
	s.Write8(c.tima)
	s.Write8(c.tma)
	s.Write8(c.tac)

	s.WriteBool(c.Enabled)
	s.Write16(c.currentBit)
	s.WriteBool(c.lastBit)
	s.WriteBool(c.overflow)
	s.Write8(c.ticksSinceOverflow)
}

func (c *Controller) AttachRegenerate(cycle func()) {
	c.cycleFunc = cycle
}
