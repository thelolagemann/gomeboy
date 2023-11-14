package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
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

	waveRAMLastRead     uint64
	waveRAMLastPosition uint8

	// embed APU to access from wave RAM Read/Write
	apu *APU
}

func newChannel3(a *APU, b *io.Bus) *channel3 {
	c := &channel3{
		channel: newChannel(),
		apu:     a,
	}
	a.s.RegisterEvent(scheduler.APUChannel3, func() {
		if c.enabled && c.dacEnabled {
			c.waveRAMPosition = (c.waveRAMPosition + 1) & 31

			c.waveRAMLastRead = a.s.Cycle()
			c.waveRAMLastPosition = c.waveRAMPosition >> 1
			c.waveRAMSampleBuffer = c.waveRAM[c.waveRAMLastPosition]
		} else {
			c.waveRAMSampleBuffer = 0
		}

		inCycles := uint64((2048 - c.frequency) * 2)
		a.s.ScheduleEvent(scheduler.APUChannel3, inCycles)
	})

	c.channelBit = types.Bit2
	b.ReserveAddress(types.NR30, whenEnabled(a, types.NR30, func(v byte) byte {
		c.dacEnabled = v&types.Bit7 != 0
		if !c.dacEnabled {
			c.enabled = false

			return 0x7F
		}

		return 0xFF
	}))
	b.ReserveAddress(types.NR31, func(v byte) byte {
		switch a.model {
		case types.CGBABC, types.CGB0:
			if a.enabled {
				c.lengthLoad = v
				c.lengthCounter = 0x100 - uint(c.lengthLoad)
			}
		default:
			c.lengthLoad = v
			c.lengthCounter = 0x100 - uint(c.lengthLoad)
		}

		return 0xFF // write only
	})
	b.ReserveAddress(types.NR32, whenEnabled(a, types.NR32, func(v byte) byte {
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

		return v | 0x9F
	}))
	b.ReserveAddress(types.NR33, whenEnabled(a, types.NR33, c.setNRx3))
	b.ReserveAddress(types.NR34, whenEnabled(a, types.NR34, func(v byte) byte {
		c.frequency = (c.frequency & 0x00FF) | (uint16(v&0x7) << 8)
		lengthCounterEnabled := v&types.Bit6 != 0
		if a.firstHalfOfLengthPeriod && !c.lengthCounterEnabled && lengthCounterEnabled && c.lengthCounter > 0 {
			c.lengthCounter--
			c.enabled = c.lengthCounter > 0
		}
		c.lengthCounterEnabled = lengthCounterEnabled
		if v&types.Bit7 != 0 {
			// handle blarrgs 10-wave trigger while on test
			if c.isEnabled() && a.s.Until(scheduler.APUChannel3) == 2 && (a.model != types.CGBABC && a.model != types.CGB0) {
				newPos := (c.waveRAMPosition + 1) & 31
				pos := newPos >> 1

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
		}

		if c.lengthCounterEnabled {
			return 0xFF
		}

		return 0xBF
	}))
	b.Set(types.NR34, 0xBF) // starting value across all devices

	return c
}

func (c *channel3) getAmplitude() uint8 {
	if c.enabled && c.dacEnabled {
		shift := 0
		if c.waveRAMPosition&1 == 0 {
			shift = 4
		}
		return ((c.waveRAMSampleBuffer >> shift) & 0x0f) >> c.volumeCodeShift
	} else {
		return 0
	}
}

func (c *channel3) readWaveRAM(address uint16) uint8 {
	//fmt.Printf("WAV RAM %04x -> %02x\n", address, c.waveRAM[address-0xFF30])
	if c.isEnabled() {
		if c.apu.s.Cycle()-c.waveRAMLastRead < 2 || c.apu.model == types.CGBABC || c.apu.model == types.CGB0 {
			return c.waveRAM[c.waveRAMLastPosition]
		} else {
			//fmt.Println(c.apu.s.Cycle(), c.waveRAMLastRead, c.apu.s.Cycle()-c.waveRAMLastRead, c.apu.s.String())
			return 0xFF
		}
	}
	return c.waveRAM[address-0xFF30]
}

func (c *channel3) writeWaveRAM(address uint16, value uint8) {
	if c.isEnabled() {
		if c.apu.s.Cycle()-c.waveRAMLastRead < 2 || c.apu.model == types.CGBABC || c.apu.model == types.CGB0 {
			c.waveRAM[c.waveRAMLastPosition] = value
		}
	} else {
		c.waveRAM[address-0xFF30] = value
	}
}
