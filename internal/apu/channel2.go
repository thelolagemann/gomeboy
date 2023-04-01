package apu

import "github.com/thelolagemann/go-gameboy/internal/types"

type channel2 struct {
	*volumeChannel

	waveDutyPosition uint8

	// NR21
	duty       uint8
	lengthLoad uint8

	// NR23/24
	frequency uint16
}

func newChannel2(a *APU) *channel2 {
	c := &channel2{}
	c2 := newChannel()
	c2.stepWaveGeneration = func() {
		c.waveDutyPosition = (c.waveDutyPosition + 1) & 0x7
	}
	c2.reloadFrequencyTimer = func() {
		c.frequencyTimer = (2048 - c.frequency) * 4
	}
	c.volumeChannel = newVolumeChannel(c2)

	types.RegisterHardware(0xff15, types.NoWrite, types.NoRead)
	types.RegisterHardware(types.NR21, writeEnabled(a, func(v uint8) {
		c.duty = (v & 0xC0) >> 6
		c.lengthLoad = v & 0x3F
		c.lengthCounter = 0x40 - uint(c.lengthLoad)
	}), func() uint8 {
		return c.duty<<6 | 0x3F
	})
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
			c.enabled = c.dacEnabled
			if c.lengthCounter == 0 {
				c.lengthCounter = 0x40
				if c.lengthCounterEnabled && a.firstHalfOfLengthPeriod {
					c.lengthCounter--
				}
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

var (
	channel2Duty = [4][8]uint8{
		{0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1, 1, 1},
		{0, 1, 1, 1, 1, 1, 1, 0},
	}
)
