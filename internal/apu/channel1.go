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

func newChannel1(a *APU, b *io.Bus) *channel1 {
	// create the higher level channel
	c := &channel1{}
	c2 := newChannel()
	c2.channelBit = types.Bit0

	b.ReserveAddress(types.NR10, didChange(a, c2, func(v byte) byte {
		if !a.enabled {
			return a.b.Get(types.NR10)
		}
		c.sweepPeriod = (v & 0x70) >> 4
		c.negate = v&types.Bit3 != 0
		c.shift = v & 0x7
		if !c.negate && c.negateHasHappened {
			c.enabled = false
		}
		return v | 0x80
	}))
	b.ReserveAddress(types.NR11, func(v byte) byte {
		if a.enabled {
			c.setDuty(v)
		}

		switch b.Model() {
		case types.CGBABC, types.CGB0:
			// CGB only sets length if APU is enabled
			if a.enabled {
				c.setLength(v)
			}
		default:
			c.setLength(v)
		}
		return c.duty<<6 | 0x3F
	})
	b.ReserveSetAddress(types.NR11, func(a any) {
		c.setDuty(a.(uint8))
		c.setLength(a.(uint8))

		b.Set(types.NR11, c.duty<<6|0x3F)
	})
	b.ReserveAddress(types.NR12, whenEnabled(a, types.NR12, didChange(a, c2, func(v byte) byte {
		c.setNRx2(v)
		return c.getNRx2()
	})))
	b.ReserveAddress(types.NR13, whenEnabled(a, types.NR13, c2.setNRx3))
	b.ReserveAddress(types.NR14, didChange(a, c2, func(v byte) byte {
		if !a.enabled {
			return a.b.Get(types.NR14)
		}
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

		if c.lengthCounterEnabled {
			return 0xFF
		}

		return 0xBF
	}))

	c.volumeChannel = newVolumeChannel(c2)
	a.s.RegisterEvent(scheduler.APUChannel1, func() {
		c.waveDutyPosition = (c.waveDutyPosition + 1) & 0x7
		a.s.ScheduleEvent(scheduler.APUChannel1, uint64((2048-c.frequency)*4))
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

func (c *channel1) getAmplitude() float32 {
	if c.enabled && c.dacEnabled {
		dacInput := channel1Duty[c.duty][c.waveDutyPosition] * c.currentVolume
		dacOutput := (float32(dacInput) / 7.5) - 1
		return dacOutput
	} else {
		return 0
	}
}

var (
	channel1Duty = [4][8]uint8{
		{0, 0, 0, 0, 0, 0, 0, 1}, // 12.5% duty cycle
		{1, 0, 0, 0, 0, 0, 0, 1}, // 25% duty cycle
		{1, 0, 0, 0, 0, 1, 1, 1}, // 50% duty cycle
		{0, 1, 1, 1, 1, 1, 1, 0}, // 75% duty cycle
	}
)
