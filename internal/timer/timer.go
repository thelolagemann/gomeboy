package timer

import (
	"github.com/thelolagemann/go-gameboy/internal/apu"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

// Controller is a timer controller. It is used to generate
// interrupts at a specific frequency. The frequency can be
// configured using the types.TAC register.
type Controller struct {
	currentBit uint8 // the current bit of the DIV register that is used to increment the TIMA register

	tima uint8 // types.TIMA
	tma  uint8 // types.TMA
	tac  uint8 // types.TAC

	irq *interrupts.Service
	s   *scheduler.Scheduler

	reloading     bool
	reloadPending bool
	reloadCancel  bool
	enabled       bool
}

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service, s *scheduler.Scheduler, a *apu.APU) *Controller {
	c := &Controller{
		irq: irq,
		s:   s,
	}

	// set up events
	s.RegisterEvent(scheduler.TimerTIMAIncrement, c.scheduledTIMAIncrement)
	s.RegisterEvent(scheduler.TimerTIMAReload, c.reloadTIMA)
	s.RegisterEvent(scheduler.TimerTIMAFinishReload, func() {
		c.reloading = false
	})

	// set up registers
	types.RegisterHardware(
		types.DIV,
		func(v uint8) {
			// writing to DIV resets the counter to 0, so the TIMA
			// could also be affected by a falling edge, if the selected bit
			// of DIV is 1, as a falling edge would be detected as DIV gets
			// reset to 0

			// calculate internal div
			internal := s.SysClock()

			// check for an abrupt increment caused by the div reset
			if internal&timerBits[c.currentBit] != 0 && c.enabled { // we don't need to check the new value, because it's always 0
				// visualization just in case you're still confused:
				//
				// Current Bit: 3 (0b1000) 16 cycles
				//
				// DIV 0b0000_1100 => 0b0000_0000
				//            ^ ------------ ^ = falling edge
				c.abruptlyIncrementTIMA()
			}

			// update the last cycle
			c.s.SysClockReset()
			// TODO APU frame sequencer is tied to the DIV register
			// in double speed, if bit 5 of DIV is 1, the APU frame sequencer
			// will advance, in normal speed if bit 4 of DIV is 1 the APU frame
			// sequencer will advance. again, we don't need to check the new value
			// because it's always 0 so a falling edge will always be detected
			// the frame sequencer should then be scheduled to advance again
			// after 8192 cycles
			if internal&0b1_0000 != 0 {
				// TODO schedule APU frame sequencer
				a.StepFrameSequencer()
			}

			// reschedule APU frame sequencer
			s.DescheduleEvent(scheduler.APUFrameSequencer)
			s.ScheduleEvent(scheduler.APUFrameSequencer, 8192)

			// the internal timer uses the same clock as the DIV register
			// so a write to DIV will also reset the internal timer, which
			// means we to need to reschedule the timer increment event
			// to prevent the timer from incrementing too fast
			// https://github.com/Gekkio/mooneye-test-suite/blob/main/acceptance/timer/div_write.s
			s.DescheduleEvent(scheduler.TimerTIMAIncrement)
			s.DescheduleEvent(scheduler.TimerTIMAReload)
			s.DescheduleEvent(scheduler.TimerTIMAFinishReload)
			c.s.DescheduleEvent(scheduler.SerialBitTransfer)
			c.s.ScheduleEvent(scheduler.SerialBitTransfer, 512)

			s.ScheduleEvent(scheduler.TimerTIMAIncrement, timaCycles[c.currentBit])
		}, func() uint8 {
			return uint8(c.s.SysClock() >> 8)
		},
		types.WithSet(func(v interface{}) {
			s.OverrideDiv(v.(uint16))
		}))

	types.RegisterHardware(
		types.TIMA,
		func(v uint8) {
			// if you write to TIMA the same tick that TIMA is reloading
			// TIMA will be set to the value of TMA
			if !c.reloading {
				c.tima = v
			} else {
				c.tima = c.tma
			}
			if c.reloadPending {
				c.reloadCancel = true
			}
		}, func() uint8 {
			return c.tima
		},
	)
	types.RegisterHardware(
		types.TMA,
		func(v uint8) {
			c.tma = v

			// if you write to TMA the same tick that TIMA is reloading
			// TIMA will be set to the new value of TMA
			if c.reloading {
				c.tima = v
			}
		}, func() uint8 {
			return c.tma
		},
	)
	types.RegisterHardware(
		types.TAC,
		func(v uint8) {
			oldBit := c.currentBit
			c.changeSpeed(v & 0b11)

			// disabling the timer could cause an abrupt increment
			// if the selected bit of DIV is 1, as disabling the timer
			// will disconnect the DIV register from the timer, thus
			// causing a falling edge to be detected
			if c.enabled && v&types.Bit2 == 0 {
				if c.s.SysClock()&timerBits[oldBit] != 0 {
					c.abruptlyIncrementTIMA()
				}
			}

			// update enabled flag
			c.enabled = v&types.Bit2 == types.Bit2
		}, func() uint8 {
			return c.tac | 0b11111000
		},
	)

	return c
}

var timaCycles = [4]uint64{
	1024,
	16,
	64,
	256,
}

// reloadTIMA is called by the scheduler when the timer
// should be reloaded.
func (c *Controller) reloadTIMA() {
	// acknowledge reload
	c.reloadPending = false

	// if the reload was not cancelled, set the timer to the new value
	if !c.reloadCancel {
		c.tima = c.tma
		c.irq.Request(interrupts.TimerFlag)
		c.reloadCancel = false // reset cancel flag
	}

	// set reloading flag & schedule finish reload
	c.reloading = true
	c.s.ScheduleEvent(scheduler.TimerTIMAFinishReload, 1)
}

// abruptlyIncrementTIMA is called when conditions are met
// that would cause an abrupt increment of the timer.
func (c *Controller) abruptlyIncrementTIMA() {
	c.tima++

	// an abrupt increment that causes a reload is performed
	// instantly, rather than being delayed by 1-M cycle
	if c.tima == 0 {
		c.tima = c.tma
		c.irq.Request(interrupts.TimerFlag)
	}
}

// scheduledTIMAIncrement is called by the scheduler when the timer
// should increment.
func (c *Controller) scheduledTIMAIncrement() {
	if c.enabled {
		c.tima++

		// if the timer overflows, reload it
		if c.tima == 0 {

			// set reload pending
			c.reloadPending = true

			// schedule TIMA reload
			c.s.ScheduleEvent(scheduler.TimerTIMAReload, 4)
		}
	}
	c.s.ScheduleEvent(scheduler.TimerTIMAIncrement, timaCycles[c.currentBit])
}

// changeSpeed changes the speed of the timer, rescheduling
// any events that are affected by the change.
func (c *Controller) changeSpeed(newBit uint8) {
	internal := c.s.SysClock()

	// changing the speed could cause an abrupt increment if the
	// currently selected bit of DIV is 1, and the new bit is 0
	if internal&timerBits[c.currentBit] != 0 && internal&timerBits[newBit] == 0 {
		c.abruptlyIncrementTIMA()
	}

	ticksUntilIncrement := uint16(timaCycles[newBit]) - (internal & uint16(timaCycles[newBit]-1))
	c.s.DescheduleEvent(scheduler.TimerTIMAReload)
	c.s.DescheduleEvent(scheduler.TimerTIMAFinishReload)
	c.s.DescheduleEvent(scheduler.TimerTIMAIncrement)
	c.s.DescheduleEvent(scheduler.SerialBitTransfer)
	c.s.ScheduleEvent(scheduler.SerialBitTransfer, 512)
	c.s.ScheduleEvent(scheduler.TimerTIMAIncrement, uint64(ticksUntilIncrement))

	c.currentBit = newBit
}

// timerBits is a lookup table for the bits of the DIV register
// that are used by each timer speed. For example, if bit 9 is
// set, then the timer will increment every 1024 cycles, as that is when
// bit 9 of the DIV register would cause a falling edge. Here are some
// examples to help visualize this:
//
// Bit 9: (1024 cycles)
//
//	Cycle 1023 (0b0011_1111_1111) -> 1024 (0b0100_0000_0000)
//	                ^ ------------------------ ^ = falling edge
//	Cycle 2047 (0b0111_1111_1111) -> 2048 (0b1000_0000_0000)
//	                ^ ------------------------ ^ = falling edge
//
// Bit 3: (16 cycles)
//
//	Cycle 15 (0b0000_1111) -> 16 (0b0001_0000)
//	                 ^ ----------------- ^ = falling edge
//	Cycle 31 (0b0001_1111) -> 32 (0b0010_0000)
//	                 ^ ----------------- ^ = falling edge
//
// Bit 5: (64 cycles)
//
//	Cycle  63 (0b0011_1111) ->  64 (0b0100_0000)
//	               ^ ------------------ ^ = falling edge
//	Cycle 127 (0b0111_1111) -> 128 (0b1000_0000)
//	               ^ ------------------ ^ = falling edge
//
// Bit 7: (256 cycles)
//
//	Cycle 255 (0b0000_1111_1111) -> 256 (0b0001_0000_0000)
//	                  ^ ----------------------- ^ = falling edge
//	Cycle 511 (0b0001_1111_1111) -> 512 (0b0010_0000_0000)
//	                  ^ ----------------------- ^ = falling edge
var timerBits = [4]uint16{
	// bit 9
	0b10_0000_0000,
	// bit 3
	0b0000_1000,
	// bit 5
	0b0010_0000,
	// bit 7
	0b1000_0000,
}

var _ types.Stater = (*Controller)(nil)

// Load loads the state of the controller.
func (c *Controller) Load(s *types.State) {
	c.tima = s.Read8()
	c.tma = s.Read8()
	c.tac = s.Read8()

}

// Save saves the state of the controller.
func (c *Controller) Save(s *types.State) {
	s.Write8(c.tima)
	s.Write8(c.tma)
	s.Write8(c.tac)
}
