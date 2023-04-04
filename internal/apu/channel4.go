package apu

import (
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type channel4 struct {
	*volumeChannel

	lfsr uint16

	// NR41
	lengthLoad uint8

	// NR43
	clockShift  uint8
	widthMode   uint8
	divisorCode uint8

	isScheduled bool
}

func newChannel4(a *APU) *channel4 {
	c := &channel4{
		lfsr: 0x7FFF,
	}
	c2 := newChannel()
	c2.stepWaveGeneration = func() {
		newBit := (c.lfsr & 0b01) ^ ((c.lfsr & 0b10) >> 1)
		c.lfsr >>= 1
		c.lfsr |= newBit << 14
		if c.widthMode != 0 {
			c.lfsr &^= 1 << 6
			c.lfsr |= newBit << 6
		}
	}
	c2.reloadFrequencyTimer = func() {
		inCycles := uint64(0)
		if c.divisorCode == 0 {
			inCycles = 8 << c.clockShift
		} else {
			inCycles = uint64(c.divisorCode<<4) << c.clockShift
		}
		if inCycles > 8 {
			a.s.ScheduleEvent(scheduler.APUChannel4, inCycles)
			c.isScheduled = true
		} else {
			c.frequencyTimer = uint16(inCycles)
			c.isScheduled = false
		}
	}
	c.volumeChannel = newVolumeChannel(c2)
	a.s.RegisterEvent(scheduler.APUChannel4, c.step)

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
		c.widthMode = (v & types.Bit3) >> 3
		c.divisorCode = v & 0x7
	}), func() uint8 {
		return c.clockShift<<4 | c.widthMode<<3 | c.divisorCode
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
			c.isScheduled = false
			c.reloadFrequencyTimer()

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

func (c *channel4) step() {
	c.frequencyTimer--
	if c.frequencyTimer == 0 {
		c.reloadFrequencyTimer()
		c.stepWaveGeneration()
	}
}
