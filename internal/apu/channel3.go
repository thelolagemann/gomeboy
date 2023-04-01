package apu

import "github.com/thelolagemann/go-gameboy/internal/types"

type channel3 struct {
	*channel
	waveRAM             [16]uint8
	waveRAMPosition     uint8
	waveRAMSampleBuffer uint8

	// NR31
	lengthLoad uint8

	// NR32
	volumeCode      uint8
	volumeCodeShift uint8

	// NR33/34
	frequency uint16

	ticksSinceRead uint8
}

func newChannel3(a *APU) *channel3 {
	c := &channel3{
		channel: newChannel(),
	}

	c.channel.reloadFrequencyTimer = func() {
		c.frequencyTimer = (2048 - c.frequency) * 2
	}

	types.RegisterHardware(types.NR30, writeEnabled(a, func(v uint8) {
		c.dacEnabled = v&types.Bit7 != 0
		c.enabled = c.dacEnabled
	}), func() uint8 {
		b := uint8(0)
		if c.channel.dacEnabled {
			b |= types.Bit7
		}
		return b | 0x7F
	})
	types.RegisterHardware(types.NR31, writeEnabled(a, func(v uint8) {
		c.lengthLoad = v
		c.lengthCounter = 0x100 - uint(c.lengthLoad)
	}), func() uint8 {
		return 0xFF // write only
	})
	types.RegisterHardware(types.NR32, writeEnabled(a, func(v uint8) {
		c.volumeCode = (v & 0x60) >> 5
		switch c.volumeCode {
		case 0b00:
			c.volumeCodeShift = 4
		case 0b01:
			c.volumeCodeShift = 0
		case 0b10:
			c.volumeCodeShift = 1
		case 0b11:
			c.volumeCodeShift = 2
		}
	}), func() uint8 {
		return c.volumeCode<<5 | 0x9F
	})
	types.RegisterHardware(types.NR33, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x700) | uint16(v)
	}), func() uint8 {
		return 0xFF // write only
	})
	types.RegisterHardware(types.NR34, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x00FF) | (uint16(v&0x7) << 8)
		lengthCounterEnabled := v&types.Bit6 != 0
		if a.firstHalfOfLengthPeriod && !c.lengthCounterEnabled && lengthCounterEnabled && c.lengthCounter > 0 {
			c.lengthCounter--
			c.enabled = c.lengthCounter > 0
		}
		c.lengthCounterEnabled = lengthCounterEnabled
		if v&types.Bit7 != 0 {
			// trigger
			c.enabled = c.dacEnabled
			if c.lengthCounter == 0 {
				c.lengthCounter = 0x100
				if c.lengthCounterEnabled && a.firstHalfOfLengthPeriod {
					c.lengthCounter--
				}
			}
			c.waveRAMPosition = 0
			c.frequencyTimer = (2048-c.frequency)*2 + 6 // + 6 to pass blargg's 09-wave read while on test
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

func (c *channel3) step() {
	c.ticksSinceRead++

	if c.frequencyTimer--; c.frequencyTimer == 0 {
		c.frequencyTimer = (2048 - c.frequency) * 2
		c.ticksSinceRead = 0
		c.waveRAMPosition = (c.waveRAMPosition + 1) % 32
		c.waveRAMSampleBuffer = c.waveRAM[c.waveRAMPosition/2]
	}
}

func (c *channel3) readWaveRAM(address uint16) uint8 {
	if c.isEnabled() {
		if c.ticksSinceRead < 2 {
			return c.waveRAM[c.waveRAMPosition/2]
		} else {
			return 0xFF
		}
	} else {
		return c.waveRAM[address-0xFF30]
	}
}

func (c *channel3) writeWaveRAM(address uint16, value uint8) {
	if c.isEnabled() {
		if c.ticksSinceRead < 2 {
			c.waveRAM[c.waveRAMPosition/2] = value
		}
	} else {
		c.waveRAM[address-0xFF30] = value
	}
}
