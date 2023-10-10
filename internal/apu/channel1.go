package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

type channel1 struct {
	*volumeChannel
	// NR10
	sweepPeriod       uint8
	negate            bool
	shift             uint8
	sweepTimer        uint8
	frequencyShadow   uint16
	sweepEnabled      bool
	negateHasHappened bool
}

func writeEnabled(a *APU, f func(v uint8)) func(v uint8) {
	return func(v uint8) {
		if a.enabled {
			f(v)
		}
	}
}

func newChannel1(a *APU, b *io.Bus) *channel1 {
	// create the higher level channel
	c := &channel1{}
	c2 := newChannel()

	c.volumeChannel = newVolumeChannel(c2)
	a.s.RegisterEvent(scheduler.APUChannel1, func() {
		c.waveDutyPosition = (c.waveDutyPosition + 1) & 0x7
		a.s.ScheduleEvent(scheduler.APUChannel1, uint64((2048-c.frequency)*4))
	})

	types.RegisterHardware(types.NR10, writeEnabled(a, func(v uint8) {
		c.sweepPeriod = (v & 0x70) >> 4
		c.negate = v&types.Bit3 != 0
		c.shift = v & 0x7
		if !c.negate && c.negateHasHappened {
			c.enabled = false
		}
	}), func() uint8 {
		b := (c.sweepPeriod << 4) | (c.shift)
		if c.negate {
			b |= types.Bit3
		}
		return b | 0x80
	})
	types.RegisterHardware(types.NR11, func(v uint8) {
		if a.enabled {
			c.setDuty(v)
		}

		switch a.model {
		case types.CGBABC, types.CGB0:
			if a.enabled {
				c.setLength(v)
			}
		case types.DMGABC, types.DMG0:
			c.setLength(v)
		}
	}, c.getNRx1, registerSetter(func(v interface{}) {
		c.setDuty(v.(uint8))
		c.setLength(v.(uint8))
	}))
	types.RegisterHardware(types.NR12, writeEnabled(a, c.setNRx2), c.getNRx2, registerSetter(func(v interface{}) {
		c.setNRx2(v.(uint8))
	}))

	types.RegisterHardware(types.NR13, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x700) | uint16(v)
	}), func() uint8 {
		return 0xFF // write only
	})
	types.RegisterHardware(types.NR14, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x00FF) | ((uint16(v) & 0x07) << 8)
		lengthCounterEnabled := v&types.Bit6 != 0
		// obscure length counter behavior (see https://gbdev.gg8.se/wiki/articles/Gameboy_sound_hardware#Length_Counter)
		if a.firstHalfOfLengthPeriod && !c.lengthCounterEnabled && lengthCounterEnabled && c.lengthCounter > 0 {
			c.lengthCounter--
			c.enabled = c.lengthCounter > 0
		}
		c.lengthCounterEnabled = lengthCounterEnabled
		trigger := v&types.Bit7 != 0
		if trigger {
			c.enabled = c.dacEnabled

			if c.lengthCounter == 0 {
				c.lengthCounter = 0x40
				if c.lengthCounterEnabled && a.firstHalfOfLengthPeriod {
					c.lengthCounter--
				}
			}
			// deschedule the current event
			a.s.DescheduleEvent(scheduler.APUChannel1)
			// schedule the next event
			a.s.ScheduleEvent(scheduler.APUChannel1, uint64((2048-c.frequency)*4))

			c.initVolumeEnvelope()
			c.frequencyShadow = c.frequency
			if c.sweepPeriod > 0 {
				c.sweepTimer = c.sweepPeriod
			} else {
				c.sweepTimer = 8
			}
			c.sweepEnabled = c.sweepPeriod > 0 || c.shift > 0
			c.negateHasHappened = false
			if c.shift > 0 {
				c.frequencyCalculation()
			}
		}
	}), func() uint8 {
		b := uint8(0)
		if c.lengthCounterEnabled {
			b |= types.Bit6
		}
		return b | 0xBF
	})

	return c
}

func newVolumeChannel(channel *channel) *volumeChannel {
	return &volumeChannel{
		channel: channel,
	}
}

func (c *channel1) sweepClock() {
	if c.sweepTimer > 0 {
		c.sweepTimer--
	}
	if c.sweepTimer == 0 {
		if c.sweepPeriod > 0 {
			c.sweepTimer = c.sweepPeriod
		} else {
			c.sweepTimer = 8
		}
		if c.sweepEnabled && c.sweepPeriod > 0 {
			calculated := c.frequencyCalculation()
			if calculated <= 0x07FF && c.shift > 0 {
				c.frequencyShadow = calculated
				c.frequency = calculated
				c.frequencyCalculation()
			}
		}
	}
}

func (c *channel1) frequencyCalculation() uint16 {
	calculated := c.frequencyShadow >> c.shift
	if c.negate {
		calculated = c.frequencyShadow - 1*calculated
	}
	calculated += c.frequencyShadow
	if calculated > 0x07FF {
		c.enabled = false
	}
	c.negateHasHappened = c.negate
	return calculated
}

func (c *channel1) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		return channel1Duty[c.duty][c.waveDutyPosition] * c.currentVolume
	} else {
		return 0
	}
}

var (
	channel1Duty = [256][256]uint8{
		{0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1, 1, 1},
		{0, 1, 1, 1, 1, 1, 1, 0},
	}
)
