// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
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

// Controller is the controller for the timer. It has four registers:
//
//   - DividerRegister: The divider register. It is incremented at a rate of 16384Hz.
//   - CounterRegister: The counter register. It is incremented at a rate specified by the control register.
//   - ModuloRegister: The modulo register. When the counter overflows, it is reset to the value of this register.
//   - ControlRegister: The control register. It specifies the timer frequency.
type Controller struct {
	divider uint16 // the divider register

	counter uint8 // the counter register (TIMA)
	modulo  uint8 // the modulo register (TMA)
	control uint8 // the control register (TAC)

	overflowing bool // true if the timer overflowed during the last cycle

	irq *interrupts.Service // the interrupt controller
}

// NewController returns a new controller.
func NewController(irq *interrupts.Service) *Controller {
	return &Controller{
		divider: 0,
		counter: 0,
		modulo:  0,
		control: 0,
		irq:     irq,
	}
}

// Read returns the value of the register at the specified address.
func (c *Controller) Read(address uint16) uint8 {
	switch address {
	case DividerRegister:
		return uint8(c.divider >> 8)
	case CounterRegister:
		return c.counter
	case ModuloRegister:
		return c.modulo
	case ControlRegister:
		return c.control & 0b111
	}

	panic(fmt.Sprintf("timer: illegal read from address 0x%04X", address))
}

// Write writes the value to the register at the specified address.
func (c *Controller) Write(address uint16, value uint8) {
	switch address {
	case DividerRegister:
		// writing to the divider register resets it
		c.divider = 0
	case CounterRegister:
		c.counter = value
	case ModuloRegister:
		c.modulo = value
	case ControlRegister:
		// only the lower 3 bits are writable
		c.control = value & 0b111
	default:
		panic(fmt.Sprintf("timer: illegal write to address 0x%04X", address))
	}
}

// Step steps the timer by the specified number of cycles.
func (c *Controller) Step(cycles uint8) {
	// update DIV 16 bit value (always incrementing at 16384Hz)
	c.divider += uint16(cycles)

	// handle delayed IRQ
	if c.irq.Enabling {
		c.irq.Enabling = false
		c.irq.IME = true
	}

	// handle TIMA overflow during last cycle
	if c.overflowing {
		c.overflowing = false
		c.counter = c.modulo
		c.irq.Request(interrupts.TimerFlag)
	}

	if c.isEnabled() {
		if c.counter == 0xFF {
			c.overflowing = true
			c.divider = 0
		} else {
			// update the timer's value at certain frequencies (specified by the control register)
			freq := c.getMultiplexerMask()
			c.counter += uint8(((c.divider + uint16(cycles)) / freq) - (c.divider / freq))
		}
	}
}

// isEnabled returns true if the timer is enabled.
func (c *Controller) isEnabled() bool {
	return c.control&0x4 > 0
}

// getMultiplexerMask returns the multiplexer mask.
func (c *Controller) getMultiplexerMask() uint16 {
	switch c.control & 0x3 {
	case 0:
		return 0x200
	case 3:
		return 0x80
	case 2:
		return 0x20
	case 1:
		return 0x8
	}
	panic("timer: invalid multiplexer mask")
}
