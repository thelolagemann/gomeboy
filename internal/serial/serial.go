package serial

import (
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
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
	data    uint8 // holds the data register, AKA types.SB.
	control uint8 // holds the control register, AKA types.SC.

	count           uint8 // the number of bits that have been transferred.
	InternalClock   bool  // if true, this controller is the master.
	TransferRequest bool  // if true, a transfer has been requested.

	irq            *interrupts.Service // the interrupt service.
	AttachedDevice Device              // the device that is attached to this controller.

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
func NewController(irq *interrupts.Service, s *scheduler.Scheduler) *Controller {
	c := &Controller{
		irq:            irq,
		AttachedDevice: nullDevice{},
		s:              s,
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

		// was the transfer request bit set?
		if c.TransferRequest {
			// we need to determine when to schedule the first bit transfer,
			// when bit 8 of DIV produces a falling edge.
			// e.g.
			// DIV = 0b0000_0001_0000_0000 (512)
			// DIV = 0b0000_0001_0000_0001 (513)
			// ...
			// DIV = 0b0000_0010_0000_0000 (1024)

			// a bit is sent every 128 M-cycles (8.192 kHz)
			ticksToGo := s.SysClock() & (ticksPerBit - 1)
			s.ScheduleEvent(scheduler.SerialBitTransfer, uint64(ticksPerBit-ticksToGo))
		}
	}, func() uint8 {
		return c.control | 0x7E
	})

	s.RegisterEvent(scheduler.SerialBitTransfer, func() {
		var bit bool
		if c.AttachedDevice != nil {
			bit = c.AttachedDevice.Send()
			c.AttachedDevice.Receive(c.data&types.Bit7 == types.Bit7)
		}

		c.data = c.data << 1
		if bit {
			c.data |= 1
		}

		c.count++
		if c.count == 8 {
			c.count = 0
			c.TransferRequest = false
			c.control &^= types.Bit7
		} else if c.count == 7 {
			// schedule interrupt to happen 1 cycle before count reaches 8 (TODO find out why, possibly the CPU interrupt handling?)
			s.ScheduleEvent(scheduler.SerialBitInterrupt, ticksPerBit-4)
		} else {
			ticksToGo := s.SysClock() & (ticksPerBit - 1)
			s.ScheduleEvent(scheduler.SerialBitTransfer, uint64(ticksPerBit-ticksToGo))
		}
	})
	s.RegisterEvent(scheduler.SerialBitInterrupt, func() {
		c.irq.Request(interrupts.SerialFlag)
	})
	return c
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
	c.data = s.Read8()
	c.control = s.Read8()

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
	s.Write8(c.data)
	s.Write8(c.control)

	s.WriteBool(c.TransferRequest)
	s.Write8(c.count)
	s.WriteBool(c.InternalClock)
}
