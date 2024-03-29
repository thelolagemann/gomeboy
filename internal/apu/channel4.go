package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
)

type channel4 struct {
	*volumeChannel

	a *APU
	// NR41
	lengthLoad uint8

	// NR43
	clockShift uint8

	widthMode   bool
	divisorCode uint8
	isScheduled bool

	lfsr        uint16
	lastCatchup uint64
}

func newChannel4(a *APU, b *io.Bus) *channel4 {
	c := &channel4{
		lfsr: 0x7FFF,
		a:    a,
	}
	c2 := newChannel()
	c2.channelBit = types.Bit3
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
		c.catchup()
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
		c.catchup()
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

	return c
}

func stepShortLFSR(lfsr uint16) uint16 {
	newBit := (lfsr & 1) ^ ((lfsr & 2) >> 1)
	lfsr |= newBit << 15
	lfsr &^= 1 << 7
	lfsr |= newBit << 7
	lfsr >>= 1
	return lfsr
}
func stepLFSR(lfsr uint16) uint16 {
	newBit := (lfsr & 1) ^ ((lfsr & 2) >> 1)
	lfsr |= newBit << 15
	lfsr >>= 1
	return lfsr
}

func getTransitionValue(lfsr uint16, step uint64, short bool) uint16 {
	if short {
		for i := uint64(0); i < step; i++ {
			lfsr = stepShortLFSR(lfsr)
		}
	} else {
		for i := uint64(0); i < step; i++ {
			lfsr = stepLFSR(lfsr)
		}
	}
	return lfsr
}

func (c *channel4) catchup() {
	if c.lastCatchup > c.a.s.Cycle() {
		c.lastCatchup = c.a.s.Cycle()
		return // TODO find out root cause
	}

	// determine how many steps we should perform
	steps := (c.a.s.Cycle() - c.lastCatchup) / c.frequencyTimer

	c.lfsr = getTransitionValue(c.lfsr, steps, c.widthMode)

	// how many cycles do we have left to catch up?
	c.lastCatchup = c.a.s.Cycle() - (c.a.s.Cycle()-c.lastCatchup)%c.frequencyTimer
}

func (c *channel4) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		c.catchup()
		return (uint8(c.lfsr) & 1) * c.currentVolume
	} else {
		return 0
	}
}
