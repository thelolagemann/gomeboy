package apu

import (
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

type channel4 struct {
	*volumeChannel

	lfsr uint16

	// NR41
	lengthLoad uint8

	// NR43
	clockShift  uint8
	widthMode   bool
	divisorCode uint8

	isScheduled bool
}

func newChannel4(a *APU) *channel4 {
	c := &channel4{
		lfsr: 0x7FFF,
	}
	c2 := newChannel()

	c.volumeChannel = newVolumeChannel(c2)
	c.frequencyTimer = 8 // TODO figure out correct starting value (good enough for now)
	a.s.RegisterEvent(scheduler.APUChannel4, func() {
		// step the LFSR
		newBit := (c.lfsr & 0b01) ^ ((c.lfsr & 0b10) >> 1)
		c.lfsr >>= 1
		c.lfsr |= newBit << 14
		if c.widthMode {
			c.lfsr &^= 1 << 6
			c.lfsr |= newBit << 6
		}

		a.s.ScheduleEvent(scheduler.APUChannel4, c.frequencyTimer)
	})

	types.RegisterHardware(0xFF1F, types.NoWrite, types.NoRead)
	types.RegisterHardware(types.NR41, func(v uint8) {
		switch a.model {
		case types.CGBABC:
			if a.enabled {
				c.lengthLoad = v & 0x3F
				c.lengthCounter = 0x40 - uint(c.lengthLoad)
			}
		case types.DMGABC, types.DMG0:
			c.lengthLoad = v & 0x3F
			c.lengthCounter = 0x40 - uint(c.lengthLoad)
		}
	}, func() uint8 {
		return 0xFF // write only
	})
	types.RegisterHardware(types.NR42, writeEnabled(a, c.setNRx2), c.getNRx2)
	types.RegisterHardware(types.NR43, writeEnabled(a, func(v uint8) {
		c.clockShift = v >> 4
		c.widthMode = v&types.Bit3 != 0
		c.divisorCode = v & 0x7

		if c.divisorCode == 0 {
			c.frequencyTimer = 8 << c.clockShift
		} else {
			c.frequencyTimer = uint64(c.divisorCode<<4) << c.clockShift
		}
	}), func() uint8 {
		v := uint8(0)
		v |= c.clockShift<<4 | c.divisorCode
		if c.widthMode {
			v |= types.Bit3
		}
		return v
	})
	types.RegisterHardware(types.NR44, writeEnabled(a, func(v uint8) {
		lengthCounterEnabled := v&types.Bit6 != 0
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

			// reload frequency timer
			a.s.DescheduleEvent(scheduler.APUChannel4)
			a.s.ScheduleEvent(scheduler.APUChannel4, c.frequencyTimer)

			c.initVolumeEnvelope()
			// reset LFSR
			c.lfsr = 0x7FFF
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

func (c *channel4) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		return uint8(c.lfsr&0b1) ^ 0b1*c.currentVolume
	} else {
		return 0
	}
}
