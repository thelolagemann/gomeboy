package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

type channel2 struct {
	*volumeChannel
}

func newChannel2(a *APU, b *io.Bus) *channel2 {
	c := &channel2{}
	c2 := newChannel()

	b.ReserveAddress(types.NR21, func(v byte) byte {
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

		return c.duty<<6 | 0x3F
	})
	b.ReserveAddress(types.NR22, didChange(a, c2, func(v byte) byte {
		if !a.enabled {
			return a.b.Get(types.NR22)
		}

		c.setNRx2(v)
		return c.getNRx2()
	}))
	b.ReserveAddress(types.NR23, whenEnabled(a, types.NR23, c2.setNRx3))
	b.ReserveAddress(types.NR24, didChange(a, c2, func(v byte) byte {
		if !a.enabled {
			return a.b.Get(types.NR24)
		}

		c.frequency = (c.frequency & 0x00FF) | (uint16(v&0x7) << 8)
		lengthCounterEnabled := v&types.Bit6 != 0
		if a.firstHalfOfLengthPeriod && !c.lengthCounterEnabled && lengthCounterEnabled && c.lengthCounter > 0 {
			c.lengthCounter--
			c.enabled = c.lengthCounter > 0
		}
		c.lengthCounterEnabled = lengthCounterEnabled
		trigger := v&types.Bit7 != 0
		if trigger {
			if c.dacEnabled {
				c.enabled = true
			}

			// init length counter
			if c.lengthCounter == 0 {
				c.lengthCounter = 0x40
				if c.lengthCounterEnabled && a.firstHalfOfLengthPeriod {
					c.lengthCounter--
				}
			}

			// init frequency timer
			a.s.DescheduleEvent(scheduler.APUChannel2)
			a.s.ScheduleEvent(scheduler.APUChannel2, uint64((2048-c.frequency)*4))

			c.initVolumeEnvelope()
		}

		if c.lengthCounterEnabled {
			return 0xFF
		}

		return 0xBF
	}))

	c.volumeChannel = newVolumeChannel(c2)
	a.s.RegisterEvent(scheduler.APUChannel2, func() {
		c.waveDutyPosition = (c.waveDutyPosition + 1) & 0x7
		a.s.ScheduleEvent(scheduler.APUChannel2, uint64((2048-c.frequency)*4))
	})

	return c
}

func (c *channel2) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		return channel2Duty[c.duty][c.waveDutyPosition] * c.currentVolume
	} else {
		return 0
	}
}

var (
	channel2Duty = [256][256]uint8{
		{0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1, 1, 1},
		{0, 1, 1, 1, 1, 1, 1, 0},
	}
)
