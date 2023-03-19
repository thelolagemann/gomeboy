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
	InternalClock   bool  // if true, this controller is the master.
	TransferRequest bool  // if true, a transfer has been requested.

	irq               *interrupts.Service // the interrupt service.
	AttachedDevice    Device              // the device that is attached to this controller.
	resultFallingEdge bool                // the result of the last falling edge. (Bit 8 of DIV: 8.192 kHz)

	cycleFunc func()
}

func (c *Controller) Attach(d Device) {
	c.AttachedDevice = d
}

func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq: irq,
		// AttachedDevice: nullDevice{},
	}
	types.RegisterHardware(types.SB, func(v uint8) {
		c.data = v
	}, func() uint8 {
		return c.data
	})
	types.RegisterHardware(types.SC, func(v uint8) {
		if c.control == v|0x7E {
			return
		}
		c.control = v | 0x7E // bits 1-6 are always set
		c.InternalClock = (v & types.Bit0) == types.Bit0
		c.TransferRequest = (v & types.Bit7) == types.Bit7
		c.cycleFunc()
	}, func() uint8 {
		return c.control | 0x7E
	})
	return c
}

// TickM ticks the serial controller.
func (c *Controller) TickM(div uint16) {
	for i := 0; i < 4; i++ {
		div++
		if c.resultFallingEdge && !c.getFallingEdge(div) {
			var bit bool
			if c.AttachedDevice != nil {
				bit = c.AttachedDevice.Send()
				c.AttachedDevice.Receive(c.data&types.Bit7 == types.Bit7)
			}

			c.data = c.data << 1
			if bit {
				c.data |= 1
			}

			c.checkTransfer()
		}
		c.resultFallingEdge = c.getFallingEdge(div)
	}
}

func (c *Controller) checkTransfer() {
	if c.count++; c.count == 8 {
		c.count = 0
		c.irq.Request(interrupts.SerialFlag)

		// clear transfer request
		c.control &^= types.Bit7
		c.TransferRequest = false
	}
}

func (c *Controller) Send() bool {
	// if c is nil, or this is the master, return true.
	if c == nil || c.InternalClock {
		return true
	}
	return (c.data & types.Bit7) == types.Bit7
}

func (c *Controller) Receive(bit bool) {
	if c == nil {
		return
	}
	if !c.InternalClock {
		c.data = c.data << 1
		if bit {
			c.data |= 1
		}
		c.checkTransfer()
	}
}

func (c *Controller) HasDevice() bool {
	return c.AttachedDevice != nil
}

// getFallingEdge returns true if the falling edge of the clock is reached.
func (c *Controller) getFallingEdge(div uint16) bool {
	return ((div & (1 << 8)) != 0) && c.InternalClock && c.TransferRequest
}

var _ types.Stater = (*Controller)(nil)

// Load implements the types.Stater interface.
//
// The values are loaded in the following order:
//   - data (uint8)
//   - control (uint8)
//   - TransferRequest (bool)
//   - count (uint8)
//   - InternalClock (bool)
//   - resultFallingEdge (bool)
func (c *Controller) Load(s *types.State) {
	c.data = s.Read8()
	c.control = s.Read8()

	c.TransferRequest = s.ReadBool()
	c.count = s.Read8()
	c.InternalClock = s.ReadBool()
	c.resultFallingEdge = s.ReadBool()
}

// Save implements the types.Stater interface.
//
// The values are saved in the following order:
//   - data (uint8)
//   - control (uint8)
//   - TransferRequest (bool)
//   - count (uint8)
//   - InternalClock (bool)
//   - resultFallingEdge (bool)
func (c *Controller) Save(s *types.State) {
	s.Write8(c.data)
	s.Write8(c.control)

	s.WriteBool(c.TransferRequest)
	s.Write8(c.count)
	s.WriteBool(c.InternalClock)
	s.WriteBool(c.resultFallingEdge)
}

func (c *Controller) AttachRegenerate(cycle func()) {
	c.cycleFunc = cycle
}

// nullDevice is an implementation of Device that
// simply ignores all data.
type nullDevice struct{}

// Receive implements the Device interface.
func (n nullDevice) Receive(bool) {}

// Send implements the Device interface.
func (n nullDevice) Send() bool { return true }
