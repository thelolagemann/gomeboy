package serial

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

const (
	ticksPerBit = 512
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
	count           uint8 // the number of bits that have been transferred.
	InternalClock   bool  // if true, this controller is the master.
	TransferRequest bool  // if true, a transfer has been requested.

	b              *io.Bus
	AttachedDevice Device // the device that is attached to this controller.

	s *scheduler.Scheduler // the scheduler.
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
func NewController(b *io.Bus, s *scheduler.Scheduler) *Controller {
	c := &Controller{
		b:              b,
		AttachedDevice: nullDevice{},
		s:              s,
	}
	b.ReserveAddress(types.SB, func(v byte) byte {
		return v
	})
	b.ReserveAddress(types.SC, func(v byte) byte {
		c.InternalClock = (v & types.Bit0) == types.Bit0
		c.TransferRequest = (v & types.Bit7) == types.Bit7

		// was the transfer request bit set?
		if c.TransferRequest {
			// we need to determine when to schedule the first bit transfer,
			// when bit 8 of DIV produces a falling edge.
			// e.g.
			// DIV = 0b0000_0001_1111_1111 (511)
			// DIV = 0b0000_0010_0000_0000 (512) <- falling edge
			// DIV = 0b0000_0010_0000_0001 (513)
			// ...
			// DIV = 0b0000_0011_1111_1111 (1023)
			// DIV = 0b0000_0100_0000_0000 (1024) <- falling edge

			// is this GameBoy the master?
			if c.InternalClock {
				// a bit is sent every 128 M-cycles (8.192 kHz)
				ticksToGo := (s.SysClock() + 4) & (ticksPerBit - 1)
				s.ScheduleEvent(scheduler.SerialBitTransfer, uint64(ticksPerBit-ticksToGo))
			}
		}

		return v | 0x7E // bits 1-6 are always set
	})
	b.Set(types.SC, 0x7E) // bits 1-6 are unused

	s.RegisterEvent(scheduler.SerialBitTransfer, func() {
		if !c.InternalClock || !c.TransferRequest {
			return
		}
		bit := c.AttachedDevice.Send()
		c.AttachedDevice.Receive(c.b.Get(types.SB)&types.Bit7 == types.Bit7)

		c.b.Set(types.SB, c.b.Get(types.SB)<<1)
		if bit {
			c.b.Set(types.SB, c.b.Get(types.SB)|1)
		}

		c.count++
		if c.count == 8 {
			c.count = 0
			c.TransferRequest = false
			c.b.RaiseInterrupt(io.SerialINT)
			c.b.ClearBit(types.SC, types.Bit7)
		} else {
			ticksToGo := (s.SysClock() + 4) & (ticksPerBit - 1)
			s.ScheduleEvent(scheduler.SerialBitTransfer, uint64(ticksPerBit-ticksToGo))
		}
	})
	s.RegisterEvent(scheduler.SerialBitInterrupt, func() {
		c.b.RaiseInterrupt(io.SerialINT)
	})
	return c
}

// checkTransfer checks if a transfer has been completed, and if so,
// triggers a serial interrupt, and clears the transfer request.
func (c *Controller) checkTransfer() {
	if c.count++; c.count == 8 {
		c.count = 0
		c.b.RaiseInterrupt(io.SerialINT)

		// clear transfer request
		c.b.ClearBit(types.SC, types.Bit7)
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
	return (c.b.Get(types.SB) & types.Bit7) == types.Bit7
}

// Receive receives a bit from the attached device, and shifts it into
// the data register. If the caller is the master, it does nothing.
func (c *Controller) Receive(bit bool) {
	if !c.InternalClock {
		c.b.Set(types.SB, c.b.Get(types.SB)<<1)
		if bit {
			c.b.Set(types.SB, c.b.Get(types.SB)|1)
		}
		c.checkTransfer()
	}
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
func (c *Controller) Load(s *types.State) {
	c.TransferRequest = s.ReadBool()
	c.count = s.Read8()
	c.InternalClock = s.ReadBool()
}

// Save implements the types.Stater interface.
//
// The values are saved in the following order:
//   - data (uint8)
//   - control (uint8)
//   - TransferRequest (bool)
//   - count (uint8)
//   - InternalClock (bool)
func (c *Controller) Save(s *types.State) {
	s.WriteBool(c.TransferRequest)
	s.Write8(c.count)
	s.WriteBool(c.InternalClock)
}
