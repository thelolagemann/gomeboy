package serial

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

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

// Attach attaches a Device to the Controller.
func (c *Controller) Attach(d Device) {
	c.AttachedDevice = d
}

// NewController creates a new Controller. A Controller is responsible for
// sending and receiving data to and from devices. It is also responsible for
// triggering serial interrupts.
//
// By default, the Controller is attached to a nullDevice, which acts as if
// there is no device attached. This is the same as if the device is not
// plugged in. If you want to attach a device, use the Controller.Attach method.
func NewController(irq *interrupts.Service) *Controller {
	c := &Controller{
		irq:            irq,
		AttachedDevice: nullDevice{},
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

// TickM ticks the serial controller by 1 M-cycle. This should be called
// every M-cycle, when the serial controller is enabled.
func (c *Controller) TickM(div uint16) {
	for i := 0; i < 4; i++ {
		div++
		newEdge := c.getFallingEdge(div)
		if c.resultFallingEdge && !newEdge {
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
		c.resultFallingEdge = newEdge
	}
}

// checkTransfer checks if a transfer has been completed, and if so,
// triggers a serial interrupt, and clears the transfer request.
func (c *Controller) checkTransfer() {
	if c.count++; c.count == 8 {
		c.count = 0
		c.irq.Request(interrupts.SerialFlag)

		// clear transfer request
		c.control &^= types.Bit7
		c.TransferRequest = false
	}
}

// Send returns the leftmost bit of the data register, unless
// the caller is the master, in which case it always returns true.
// This is because the master is driving the clock, and thus should
// not be trying to read from its own data register.
func (c *Controller) Send() bool {
	// if c is the master, return true.
	if c.InternalClock {
		return true
	}
	return (c.data & types.Bit7) == types.Bit7
}

// Receive receives a bit from the attached device, and shifts it into
// the data register. If the caller is the master, it does nothing.
func (c *Controller) Receive(bit bool) {
	if !c.InternalClock {
		c.data = c.data << 1
		if bit {
			c.data |= 1
		}
		c.checkTransfer()
	}
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
