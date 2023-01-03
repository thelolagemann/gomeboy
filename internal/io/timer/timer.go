// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/pkg/bits"
)

const (
	// DividerRegister is the address of the timer divider register.
	// It is incremented at a rate specified by the TimerControlRegister.
	DividerRegister = 0xFF04
	// CounterRegister is the address of the timer counter register.
	// It is incremented at a rate specified by the TimerControlRegister.
	CounterRegister = 0xFF05
	// ModuloRegister is the address of the timer modulo register.
	// When the TimerCounterRegister overflows, it is reset to the value
	// of this register and an interrupt is requested.
	ModuloRegister = 0xFF06
	// ControlRegister is the address of the timer control register.
	// It specifies the timer frequency.
	ControlRegister = 0xFF07
)

// Controller is the controller for the timer.
type Controller struct {
	divider uint16
	counter uint8
	modulo  uint8
	control uint8

	cycles uint16
}

// NewController returns a new controller.
func NewController() *Controller {
	return &Controller{
		divider: 0,
		counter: 0,
		modulo:  0,
		control: 0,
		cycles:  0,
	}
}

// Update updates the timers. It returns true if an interrupt
// was requested.
func (c *Controller) Update(cycles uint8) bool {
	c.handleDivider(cycles)

	// the clock must be enabled to update
	if c.isEnabled() {
		c.cycles -= uint16(cycles)

		// if enough cycles have passed, update the counter
		if c.cycles <= 0 {
			// reset the timer
			c.setClockFreq()

			// if timer about to overflow
			if c.counter == 0xFF {
				// reset the counter
				c.counter = c.modulo

				// request an interrupt
				return true
			} else {
				c.counter++
			}
		}
	}

	return false
}

// isEnabled returns true if the timer is enabled.
func (c *Controller) isEnabled() bool {
	return bits.Test(c.control, 2)
}

// handleDivider handles the divider register.
func (c *Controller) handleDivider(cycles uint8) {
	c.divider += uint16(cycles)

	if c.divider >= 0xFF {
		c.divider = 0
	}
}

// getClockFreq returns the clock frequency.
func (c *Controller) getClockFreq() uint8 {
	return c.control & 0x3
}

// setClockFreq sets the clock frequency.
func (c *Controller) setClockFreq() {
	switch c.control & 0x3 {
	case 0:
		c.cycles = 1024 // 4096Hz
	case 1:
		c.cycles = 16 // 262144Hz
	case 2:
		c.cycles = 64 // 65536Hz
	case 3:
		c.cycles = 256 // 16384Hz
	}
}

// Read reads a value from the timer.
func (c *Controller) Read(addr uint16) uint8 {
	switch addr {
	case DividerRegister:
		return uint8(c.divider >> 8)
	case CounterRegister:
		return c.counter
	case ModuloRegister:
		return c.modulo
	case ControlRegister:
		return c.control
	}

	panic(fmt.Sprintf("illegal read from timer register: %X", addr))
}

// Write writes a value to the timer.
func (c *Controller) Write(addr uint16, val uint8) {
	switch addr {
	case DividerRegister:
		c.divider = 0
	case CounterRegister:
		c.counter = val
	case ModuloRegister:
		c.modulo = val
	case ControlRegister:
		currentFreq := c.getClockFreq()
		c.control = val
		if currentFreq != c.getClockFreq() {
			c.setClockFreq()
		}
	default:
		panic(fmt.Sprintf("illegal write to timer register: %X", addr))
	}
}
