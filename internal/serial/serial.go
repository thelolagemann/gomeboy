package serial

import (
	"fmt"
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

	length uint16
	count  uint8

	timer             uint16
	irq               *interrupts.Service
	attachedDevice    *Controller
	current           uint8
	resultFallingEdge bool
}

func (c *Controller) Attach(d *Controller) {
	c.attachedDevice = d
	fmt.Println("attached")
}

func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq: irq,
		//attachedDevice: &Controller{},
		current: currentSerial,
		count:   1,
		control: 0x7E,
	}
	types.RegisterHardware(types.SB, func(v uint8) {
		c.data = v
	}, func() uint8 {
		return c.data
	})
	types.RegisterHardware(types.SC, func(v uint8) {
		c.control = v | 0b0111_1110
	}, func() uint8 {
		return c.control
	})
	currentSerial++
	return c
}

// Tick ticks the serial controller.
func (c *Controller) Tick(div uint16) {
	if c.resultFallingEdge && !c.getFallingEdge(div) {
		if c.count <= 8 {
			// transfer a bit
			bit := c.attachedDevice.send()
			c.attachedDevice.receive(c.data&types.Bit7 != 0)

			// shift data
			c.data = c.data << 1
			if bit {
				c.data |= 1
			}
			c.count += 1
		}

		if c.count > 8 {
			c.count = 1
			c.control = 0x01
			c.length = 0

			// request interrupt
			c.irq.Request(interrupts.SerialFlag)
		}

	}
	c.resultFallingEdge = c.getFallingEdge(div)
}

func (c *Controller) send() bool {
	if c == nil {
		return false
	}
	fmt.Printf("send: %08b from %d\n", c.data, c.current)
	if c.internalClock() {
		return true
	}
	return (c.data & types.Bit7) == types.Bit7
}

func (c *Controller) receive(bit bool) {
	if c == nil {
		return
	}
	if bit {
		fmt.Printf("receive: %08b from %d\n", c.data, c.current, bit)
	}
	if !c.internalClock() {
		c.data = c.data << 1
		if bit {
			c.data |= 1
		}
		//c.checkTransfer()
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
