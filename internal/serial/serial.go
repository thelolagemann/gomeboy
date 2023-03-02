package serial

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

// Device is a device that can be attached to the Controller.
type Device interface {
	Receive(bool)
	Send() bool
}

// Controller is the serial controller. It is responsible for sending and
// receiving data to and from devices.
// Before a transfer, data holds the next byte to be sent. AKA types.SB
// During a transfer, it has a mix of the incoming data and the outgoing data.
// each cycle, the leftmost bit of data is sent to the attached device, and
// shifted out of data, and the incoming bit is shifted into data.
//
// example:
//
//	Before : data = o7 o6 o5 o4 o3 o2 o1 o0
//	Cycle 1: data = o6 o5 o4 o3 o2 o1 o0 i0
//	Cycle 2: data = o5 o4 o3 o2 o1 o0 i0 i1
//	Cycle 3: data = o4 o3 o2 o1 o0 i0 i1 i2
//	Cycle 4: data = o3 o2 o1 o0 i0 i1 i2 i3
//	Cycle 5: data = o2 o1 o0 i0 i1 i2 i3 i4
//	Cycle 6: data = o1 o0 i0 i1 i2 i3 i4 i5
//	Cycle 7: data = o0 i0 i1 i2 i3 i4 i5 i6
//	Cycle 8: data = i0 i1 i2 i3 i4 i5 i6 i7
//
// Where o0-o7 are the outgoing bits, and i0-i7 are the incoming bits.
type Controller struct {
	data    uint8 // holds the data register, AKA types.SB.
	control uint8 // holds the control register, AKA types.SC.

	count           uint8 // the number of bits that have been transferred.
	internalClock   bool  // if true, this controller is the master.
	transferRequest bool  // if true, a transfer has been requested.

	irq               *interrupts.Service // the interrupt service.
	attachedDevice    Device              // the device that is attached to this controller.
	resultFallingEdge bool                // the result of the last falling edge. (Bit 8 of DIV: 8.192 kHz)
}

func (c *Controller) Attach(d Device) {
	c.attachedDevice = d
}

func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq:            irq,
		attachedDevice: nullDevice{},
	}
	types.RegisterHardware(types.SB, func(v uint8) {
		c.data = v
	}, func() uint8 {
		return c.data
	})
	types.RegisterHardware(types.SC, func(v uint8) {
		c.control = v | 0x7E
		c.internalClock = (v & types.Bit0) == types.Bit0
		c.transferRequest = (v & types.Bit7) == types.Bit7
	}, func() uint8 {
		return c.control | 0x7E
	})
	return c
}

// Tick ticks the serial controller.
func (c *Controller) Tick(div uint16) {
	// is the serial transfer enabled?
	if !c.internalClock || !c.transferRequest {
		return
	}
	if c.resultFallingEdge && !c.getFallingEdge(div) {
		bit := c.attachedDevice.Send()
		c.attachedDevice.Receive(c.data&types.Bit7 == types.Bit7)

		c.data = c.data << 1
		if bit {
			c.data |= 1
		}

		c.checkTransfer()
	}
	c.resultFallingEdge = c.getFallingEdge(div)
}

func (c *Controller) checkTransfer() {
	if c.count++; c.count == 8 {
		c.count = 0
		c.irq.Request(interrupts.SerialFlag)

		// clear transfer request
		c.control &^= types.Bit7
		c.transferRequest = false
	}
}

func (c *Controller) Send() bool {
	// if c is nil, or this is the master, return true.
	if c == nil || c.internalClock {
		return true
	}
	return (c.data & types.Bit7) == types.Bit7
}

func (c *Controller) Receive(bit bool) {
	if c == nil {
		return
	}
	if !c.internalClock {
		c.data = c.data << 1
		if bit {
			c.data |= 1
		}
		c.checkTransfer()
	}
}

// getFallingEdge returns true if the falling edge of the clock is reached.
func (c *Controller) getFallingEdge(div uint16) bool {
	return ((div & (1 << 8)) != 0) && c.internalClock && c.transferRequest
}

var _ types.Stater = (*Controller)(nil)

func (c *Controller) Load(s *types.State) {
	c.data = s.Read8()
	c.control = s.Read8()

	c.transferRequest = s.ReadBool()
	c.count = s.Read8()
	c.internalClock = s.ReadBool()
	c.resultFallingEdge = s.ReadBool()
}

func (c *Controller) Save(s *types.State) {
	s.Write8(c.data)
	s.Write8(c.control)

	s.WriteBool(c.transferRequest)
	s.Write8(c.count)
	s.WriteBool(c.internalClock)
	s.WriteBool(c.resultFallingEdge)
}

type nullDevice struct{}

func (n nullDevice) Receive(bool) {}
func (n nullDevice) Send() bool   { return true }
