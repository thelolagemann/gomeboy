package apu

import (
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type channel2 struct {
	*volumeChannel
}

func newChannel2(a *APU) *channel2 {
	c := &channel2{}
	c2 := newChannel()

	c.volumeChannel = newVolumeChannel(c2)
	a.s.RegisterEvent(scheduler.APUChannel2, func() {
		c.waveDutyPosition = (c.waveDutyPosition + 1) & 0x7
		a.s.ScheduleEvent(scheduler.APUChannel2, uint64((2048-c.frequency)*4))
	})

	types.RegisterHardware(0xff15, types.NoWrite, types.NoRead)
	types.RegisterHardware(types.NR21, func(v uint8) {
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
	}, c.getNRx1)
	types.RegisterHardware(types.NR22, writeEnabled(a, c.setNRx2), c.getNRx2)
	types.RegisterHardware(types.NR23, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x700) | uint16(v)
	}), func() uint8 {
		return 0xFF // write only
	})
	types.RegisterHardware(types.NR24, writeEnabled(a, func(v uint8) {
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
	}), func() uint8 {
		b := uint8(0)
		if c.lengthCounterEnabled {
			b |= types.Bit6
		}
		return b | 0xBF
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
