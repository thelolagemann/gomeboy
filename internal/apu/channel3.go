package apu

import (
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
)

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

	waveRAMLastRead     uint64
	waveRAMLastPosition uint8
	waveRAMWriteCorrupt bool

	// embed APU to access from wave RAM Read/Write
	apu *APU

	s *scheduler.Scheduler
}

func newChannel3(a *APU) *channel3 {
	c := &channel3{
		channel:         newChannel(),
		waveRAMPosition: 4,
		apu:             a,
		s:               a.s,
	}

	a.s.RegisterEvent(scheduler.APUChannel3, func() {
		if c.enabled && c.dacEnabled {
			c.waveRAMLastRead = a.s.Cycle()
			c.waveRAMLastPosition = c.waveRAMPosition >> 1
			c.waveRAMSampleBuffer = c.waveRAM[c.waveRAMLastPosition]

			c.waveRAMPosition = (c.waveRAMPosition + 1) & 31
		} else {
			c.waveRAMSampleBuffer = 0
		}

		inCycles := uint64((2048 - c.frequency) * 2)
		a.s.ScheduleEvent(scheduler.APUChannel3, inCycles)
		//a.s.ScheduleEvent(scheduler.APUChannel3WaveRAMWriteCorruption, inCycles-2)
		// a.s.ScheduleEvent(scheduler.APUChannel3WaveRAMWriteCorruptionEnd, inCycles-1)
	})

	a.s.RegisterEvent(scheduler.APUChannel3WaveRAMWriteCorruption, func() {
		c.waveRAMWriteCorrupt = true
	})
	a.s.RegisterEvent(scheduler.APUChannel3WaveRAMWriteCorruptionEnd, func() {
		c.waveRAMWriteCorrupt = false
	})

	types.RegisterHardware(types.NR30, writeEnabled(a, func(v uint8) {
		c.dacEnabled = v&types.Bit7 != 0
		if !c.dacEnabled {
			c.enabled = false
		}
	}), func() uint8 {
		b := uint8(0)
		if c.channel.dacEnabled {
			b |= types.Bit7
		}
		return b | 0x7F
	})
	types.RegisterHardware(types.NR31, func(v uint8) {
		switch a.model {
		case types.CGBABC:
			if a.enabled {
				c.lengthLoad = v
				c.lengthCounter = 0x100 - uint(c.lengthLoad)
			}
		case types.DMGABC, types.DMG0:
			c.lengthLoad = v
			c.lengthCounter = 0x100 - uint(c.lengthLoad)
		default:
			c.lengthLoad = v
			c.lengthCounter = 0x100 - uint(c.lengthLoad)
		}
	}, func() uint8 {
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
			// handle blarrgs 10-wave trigger while on test
			if c.isEnabled() && c.waveRAMWriteCorrupt && (a.model != types.CGBABC && a.model != types.CGB0) {
				pos := c.waveRAMPosition >> 1

				if pos < 4 {
					c.waveRAM[0] = c.waveRAM[pos]
				} else {
					// align to 4 bytes
					pos &^= 3

					// copy 4 bytes
					copy(c.waveRAM[0:4], c.waveRAM[pos:pos+4])
				}
			}

			// trigger
			c.enabled = c.dacEnabled

			if c.lengthCounter == 0 {
				c.lengthCounter = 0x100
				if c.lengthCounterEnabled && a.firstHalfOfLengthPeriod {
					c.lengthCounter--
				}
			}
			c.waveRAMPosition = 0
			c.waveRAMLastPosition = 0

			a.s.DescheduleEvent(scheduler.APUChannel3)
			a.s.ScheduleEvent(scheduler.APUChannel3, uint64((2048-c.frequency)*2)+6)

			// c.frequencyTimer = 6 // + 6 to pass blargg's 09-wave read while on test
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

/*func (c *channel3) step() {
	c.ticksSinceRead++
	if c.frequencyTimer--; c.frequencyTimer == 0 {
		c.frequencyTimer = (2048 - c.frequency) * 2

		if c.enabled && c.dacEnabled {
			c.ticksSinceRead = 0
			c.waveRAMLastPosition = c.waveRAMPosition >> 1
			c.waveRAMSampleBuffer = c.waveRAM[c.waveRAMLastPosition]

			c.waveRAMPosition = (c.waveRAMPosition + 1) & 31
		} else {
			c.waveRAMSampleBuffer = 0
		}
	}
}*/

func (c *channel3) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		shift := 0
		if c.waveRAMPosition&1 == 0 {
			shift = 4
		}
		return ((c.waveRAMSampleBuffer >> shift) & 0x0F) >> c.volumeCodeShift
	} else {
		return 0
	}
}

func (c *channel3) readWaveRAM(address uint16) uint8 {
	if c.isEnabled() {
		if c.apu.model == types.CGBABC || c.apu.model == types.CGB0 {
			return c.waveRAM[c.waveRAMLastPosition]
		} else {
			return 0xFF
		}
	} else {
		return c.waveRAM[address-0xFF30]
	}
}

func (c *channel3) writeWaveRAM(address uint16, value uint8) {
	if c.isEnabled() {
		if c.apu.model == types.CGBABC || c.apu.model == types.CGB0 {
			c.waveRAM[c.waveRAMLastPosition] = value
		}
	} else {
		c.waveRAM[address-0xFF30] = value
	}
}
