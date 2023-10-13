package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
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

func newChannel4(a *APU, b *io.Bus) *channel4 {
	c := &channel4{
		lfsr: 0x7FFF,
	}
	c2 := newChannel()
	b.ReserveAddress(types.NR41, func(v byte) byte {
		switch a.model {
		case types.CGBABC, types.CGB0:
			if a.enabled {
				c.lengthLoad = v & 0x3F
				c.lengthCounter = 0x40 - uint(c.lengthLoad)
			}
		default:
			c.lengthLoad = v & 0x3F
			c.lengthCounter = 0x40 - uint(c.lengthLoad)
		}

		return 0xFF // write-only
	})
	b.ReserveAddress(types.NR42, didChange(a, c2, func(v byte) byte {
		if !a.enabled {
			return b.Get(types.NR42)
		}
		c.setNRx2(v)
		return c.getNRx2()
	}))
	b.ReserveAddress(types.NR43, func(v byte) byte {
		if !a.enabled {
			return b.Get(types.NR43)
		}
		c.clockShift = v >> 4
		c.widthMode = v&types.Bit3 != 0
		c.divisorCode = v & 0x7

		if c.divisorCode == 0 {
			c.frequencyTimer = 8 << c.clockShift
		} else {
			c.frequencyTimer = uint64(c.divisorCode<<4) << c.clockShift
		}

		return v
	})
	b.ReserveAddress(types.NR44, didChange(a, c2, func(v byte) byte {
		if !a.enabled {
			return b.Get(types.NR44)
		}
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

		if c.lengthCounterEnabled {
			return 0xff
		}

		return 0xBF
	}))
	b.Set(types.NR44, 0xBF)

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

	return c
}

func (c *channel4) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		return uint8(c.lfsr&0b1) ^ 0b1*c.currentVolume
	} else {
		return 0
	}
}
