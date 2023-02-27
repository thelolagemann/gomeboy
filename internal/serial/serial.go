package serial

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

var currentSerial uint8

// Device is a device that can be attached to the Controller.
type Device interface {
	Receive(bool)
	Send() bool
}

type Controller struct {
	data    uint8
	control uint8

	count uint8

	irq               *interrupts.Service
	attachedDevice    *Controller
	current           uint8
	resultFallingEdge bool
}

func (c *Controller) Attach(d *Controller) {
	c.attachedDevice = d
}

func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq:     irq,
		current: currentSerial,
		control: 0x7E,
	}
	types.RegisterHardware(types.SB, func(v uint8) {
		c.data = v
	}, func() uint8 {
		return c.data
	})
	types.RegisterHardware(types.SC, func(v uint8) {
		c.control = v | 0x7E
	}, func() uint8 {
		return c.control
	})
	currentSerial++
	return c
}

// Tick ticks the serial controller.
func (c *Controller) Tick(div uint16) {
	// is the serial transfer enabled?
	if !c.transferRequest() || !c.internalClock() {
		return
	}
	if c.resultFallingEdge && !c.getFallingEdge(div) {
		if c.count <= 8 {
			bit := c.attachedDevice.send()
			c.attachedDevice.receive(c.data&types.Bit7 == types.Bit7)

			c.data = c.data << 1
			if bit {
				c.data |= 1
			}
			c.checkTransfer()
		}
	}
	c.resultFallingEdge = c.getFallingEdge(div)
}

func (c *Controller) checkTransfer() {
	if c.count++; c.count == 8 {
		c.count = 0
		c.irq.Request(interrupts.SerialFlag)

		// clear transfer request
		c.control &^= types.Bit7
	}
}

func (c *Controller) send() bool {
	if c == nil {
		return false
	}
	if c.internalClock() {
		return true
	}
	return (c.data & types.Bit7) == types.Bit7
}

func (c *Controller) receive(bit bool) {
	if c == nil {
		return
	}
	if !c.internalClock() {
		c.data = c.data << 1
		if bit {
			c.data |= 1
		}
		c.checkTransfer()
	}
}

// getFallingEdge returns true if the falling edge of the clock is reached.
func (c *Controller) getFallingEdge(div uint16) bool {
	return ((div & (1 << 8)) != 0) && c.internalClock() && c.transferRequest()
}

// transferRequest returns true if a transfer is requested.
func (c *Controller) transferRequest() bool {
	return c.control&types.Bit7 != 0
}

// internalClock returns true if the internal clock is used.
func (c *Controller) internalClock() bool {
	return c.control&types.Bit0 != 0
}
