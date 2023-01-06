// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import "fmt"

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

	releaseOverflow bool // true if the counter overflowed in the last step
	overflowing     bool // true if the counter is overflowing
	fallingEdge     bool // true if the falling edge of the clock signal was detected in the last step
	carry           bool // true if the counter overflowed in the last step
}

// NewController returns a new controller.
func NewController() *Controller {
	return &Controller{
		divider: 0,
		counter: 0,
		modulo:  0,
		control: 0,
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
		return c.control
	}

	panic(fmt.Sprintf("timer: illegal read from address 0x%04X", address))
}

// Write writes the value to the register at the specified address.
func (c *Controller) Write(address uint16, value uint8) {
	switch address {
	case DividerRegister:
		c.divider = 0
		c.Step(0)
	case CounterRegister:
		if c.releaseOverflow {
			return
		}
		c.counter = value
		c.carry = false
		c.overflowing = false
		c.releaseOverflow = false
	case ModuloRegister:
		if c.releaseOverflow {
			c.counter = value
		}
		c.modulo = value
	case ControlRegister:
		c.control = value & 7
		c.Step(0)
		c.Step(0)
	default:
		panic(fmt.Sprintf("timer: illegal write to address 0x%04X", address))
	}
}

// Step steps the timer by the specified number of cycles.
func (c *Controller) Step(cycles uint8) bool {
	overflow := false
	// increment the divider register by the specified number of cycles (16384Hz)
	c.divider += uint16(cycles)

	// get the clock signal
	signal := c.divider&c.getMultiplexerMask() == c.getMultiplexerMask() && c.isEnabled()

	if c.releaseOverflow {
		// TIME: 8
		c.overflowing = false
		c.releaseOverflow = false
	}

	if c.overflowing {
		// TIME: 4
		c.counter = c.modulo
		overflow = true
		c.carry = false
		c.releaseOverflow = true
	}

	if c.detectFallingEdge(signal) {
		c.counter++
		// for a brief period (1 cycle/4 clocks) TIMA has the value 0
		if c.counter == 0x0 && c.carry {
			// TIME: 0
			c.overflowing = true
		} else if c.counter == 0x0 {
			// TIME: 0 about to overflow
			c.carry = true
		}
	}

	return overflow
}

// isEnabled returns true if the timer is enabled.
func (c *Controller) isEnabled() bool {
	return c.control&0x4 > 0
}

// detectFallingEdge detects the falling edge of the clock signal.
func (c *Controller) detectFallingEdge(signal bool) bool {
	result := !signal && c.fallingEdge
	c.fallingEdge = signal
	return result
}

// getMultiplexerMask returns the multiplexer mask.
func (c *Controller) getMultiplexerMask() uint16 {
	switch c.control & 0x3 {
	case 0:
		return 0x200
	case 1:
		return 0x80
	case 2:
		return 0x20
	case 3:
		return 0x8
	}
	panic("timer: invalid multiplexer mask")
}
