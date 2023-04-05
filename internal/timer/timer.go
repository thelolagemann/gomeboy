// Package timer provides an implementation of the Game Boy
// timer. It is used to generate interrupts at a specific
// frequency. The frequency can be configured using the
// TimerControlRegister.
package timer

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/interrupts"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

const (
	divPeriod = 16 // 16384 Hz, 32768 Hz in double speed mode
)

// Controller is a timer controller. It is used to generate
// interrupts at a specific frequency. The frequency can be
// configured using the types.TAC register.
type Controller struct {
	currentBit  uint8
	internalDiv uint8

	tima uint8
	tma  uint8
	tac  uint8

	doubleSpeed bool

	irq *interrupts.Service
	s   *scheduler.Scheduler

	externalDiv   uint8
	reloading     bool
	reloadPending bool
	reloadCancel  bool
	enabled       bool
	lastCycle     uint64
}

// NewController returns a new timer controller.
func NewController(irq *interrupts.Service, s *scheduler.Scheduler) *Controller {
	c := &Controller{
		irq:         irq,
		internalDiv: 0xAB,
		s:           s,
	}
	s.RegisterEvent(scheduler.TimerTIMAIncrement, c.scheduledTIMAIncrement)
	s.RegisterEvent(scheduler.TimerTIMAReload, c.reloadTIMA)
	s.RegisterEvent(scheduler.TimerTIMAFinishReload, func() {
		c.reloading = false
	})
	types.RegisterHardware(
		types.DIV,
		func(v uint8) {
			// reset the internal div register
			c.internalDiv = 0

			internal := (s.Cycle() - c.lastCycle) & 0xFFFF
			if internal&timerBits[c.currentBit] != 0 && c.enabled {
				c.timaIncrement(false)
			}

			c.lastCycle = s.Cycle()
			// TODO APU frame sequencer is tied to the DIV register

			// deschedule and reschedule tima increment
			s.DescheduleEvent(scheduler.TimerTIMAIncrement)
			s.DescheduleEvent(scheduler.TimerTIMAReload)
			s.DescheduleEvent(scheduler.TimerTIMAFinishReload)

			s.ScheduleEvent(scheduler.TimerTIMAIncrement, timaCycles[c.currentBit])
		}, func() uint8 {
			return uint8(uint16(((s.Cycle() - c.lastCycle) & 0xFFFF) >> 8))
		},
		types.WithSet(func(v interface{}) {

		}))

	// set up registers
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
			c.changeSpeed(v & 0b11)
			if c.enabled && v&types.Bit2 == 0 {
				fmt.Println("unexpected timer increase from disable")
				internal := (s.Cycle() - c.lastCycle) & 0xFFFF
				if internal&timerBits[c.currentBit] != 0 {
					c.timaIncrement(false)
				}
			}
			c.enabled = v&types.Bit2 == types.Bit2
		}, func() uint8 {
			v := uint8(0)
			v |= c.currentBit & 0b11
			if c.enabled {
				v |= types.Bit2
			}

			return v
		},
	)

	// c.s.ScheduleEvent(scheduler.TimerTIMAIncrement, uint64(timaCycles[c.currentBit]*2))

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
	c.s.ScheduleEvent(scheduler.TimerTIMAFinishReload, 4)
}

func (c *Controller) timaIncrement(delay bool) {
	if c.enabled {
		c.tima++

		// if the timer overflows, reload it
		if c.tima == 0 {
			// set reload pending
			c.reloadPending = true

			// schedule overflow IRQ
			if delay {
				c.s.ScheduleEvent(scheduler.TimerTIMAReload, 4) // TODO handle double speed
			} else {
				c.s.ScheduleEvent(scheduler.TimerTIMAReload, 0)
			}
		}
	}
}

// scheduledTIMAIncrement is called by the scheduler when the timer
// should increment.
func (c *Controller) scheduledTIMAIncrement() {
	c.timaIncrement(true)
	c.s.ScheduleEvent(scheduler.TimerTIMAIncrement, uint64(timaCycles[c.currentBit]))
}

func (c *Controller) changeSpeed(newBit uint8) {
	internal := (c.s.Cycle() - c.lastCycle) & 0xFFFF
	if newBit != c.currentBit {
		if (internal&timerBits[c.currentBit] == 1 && internal&timerBits[newBit] == 0) && c.enabled {
			c.timaIncrement(false)
		}

		ticksUntilIncrement := (rescheduleMasks[newBit] + 1) - (internal & rescheduleMasks[newBit])
		c.s.DescheduleEvent(scheduler.TimerTIMAIncrement)
		c.s.ScheduleEvent(scheduler.TimerTIMAIncrement, uint64(ticksUntilIncrement))
	}

	c.currentBit = newBit
}

var rescheduleMasks = [4]uint64{
	0b1111111111,
	0b1111,
	0b111111,
	0b11111111,
}
var timerBits = [4]uint64{
	9, 3, 5, 7,
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
