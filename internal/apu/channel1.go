package apu

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
)

type channel struct {
	enabled    bool
	dacEnabled bool

	// NRx1
	lengthCounter uint

	// NRx4
	frequencyTimer       uint16
	lengthCounterEnabled bool

	reloadFrequencyTimer func()
	stepWaveGeneration   func()
}

func (c *channel) step() {
	c.frequencyTimer--
	if c.frequencyTimer == 0 {
		c.reloadFrequencyTimer()
		c.stepWaveGeneration()
	}
}

type volumeChannel struct {
	*channel

	// NRx2
	startingVolume  uint8
	envelopeAddMode bool
	period          uint8

	volumeEnvelopeTimer      uint8
	currentVolume            uint8
	volumeEnvelopeIsUpdating bool
}

func (v *volumeChannel) volumeStep() {
	if v.period != 0 {
		if v.volumeEnvelopeTimer > 0 {
			v.volumeEnvelopeTimer--
			if v.volumeEnvelopeTimer == 0 {
				v.volumeEnvelopeTimer = v.period
				if v.currentVolume < 0xF && v.envelopeAddMode || v.currentVolume > 0 && !v.envelopeAddMode {
					if v.envelopeAddMode {
						v.currentVolume++
					} else {
						v.currentVolume--
					}
				} else {
					v.volumeEnvelopeIsUpdating = false
				}
			}
		}
	}
}

func (v *volumeChannel) setNRx2(v2 uint8) {
	envelopeAddMode := v2&types.Bit3 != 0

	// zombie mode glitch (see https://gbdev.gg8.se/wiki/articles/Gameboy_sound_hardware#Zombie_Mode)
	if v.enabled {
		if v.period == 0 && v.volumeEnvelopeIsUpdating || !v.envelopeAddMode {
			v.currentVolume++
		}
		if envelopeAddMode != v.envelopeAddMode {
			v.currentVolume = 0x10 - v.currentVolume
		}
		v.currentVolume &= 0x0F
	}

	v.startingVolume = v2 >> 4
	v.envelopeAddMode = envelopeAddMode
	v.period = v2 & 0x7
	v.dacEnabled = v2&0xF8 > 0
	if !v.dacEnabled {
		v.enabled = false
	}
}

func (v *volumeChannel) getNRx2() uint8 {
	b := (v.startingVolume << 4) | v.period
	if v.envelopeAddMode {
		b |= types.Bit3
	}
	return b
}

func (v *volumeChannel) initVolumeEnvelope() {
	v.volumeEnvelopeTimer = v.period
	v.currentVolume = v.startingVolume
	v.volumeEnvelopeIsUpdating = true
}

func newChannel() *channel {
	c := &channel{}

	return c
}

func (c *channel) isEnabled() bool {
	return c.enabled && c.dacEnabled
}

func (c *channel) lengthStep() {
	if c.lengthCounterEnabled && c.lengthCounter > 0 {
		c.lengthCounter--
		c.enabled = c.lengthCounter > 0
	}
}

type channel1 struct {
	// NR10
	sweepPeriod       uint8
	negate            bool
	shift             uint8
	sweepTimer        uint8
	frequencyShadow   uint16
	sweepEnabled      bool
	negateHasHappened bool

	// NR11
	duty       uint8
	lengthLoad uint8

	// NR13/14
	frequency uint16

	waveDutyPosition uint8

	*volumeChannel // lengthCounter, enabled, dacEnabled, output
}

func writeEnabled(a *APU, f func(v uint8)) func(v uint8) {
	return func(v uint8) {
		if a.enabled {
			f(v)
		}
	}
}

func newChannel1(a *APU) *channel1 {
	// create the higher level channel
	c := &channel1{}
	c2 := newChannel()
	c2.stepWaveGeneration = func() {
		c.waveDutyPosition = (c.waveDutyPosition + 1) & 0x7
	}
	c2.reloadFrequencyTimer = func() {
		c.frequencyTimer = (2048 - c.frequency) * 4
	}
	c.volumeChannel = newVolumeChannel(c2)

	types.RegisterHardware(types.NR10, writeEnabled(a, func(v uint8) {
		c.sweepPeriod = (v & 0x70) >> 4
		c.negate = v&types.Bit3 != 0
		c.shift = v & 0x7
		if !c.negate && c.negateHasHappened {
			c.enabled = false
		}
	}), func() uint8 {
		b := (c.sweepPeriod << 4) | (c.shift)
		if c.negate {
			b |= types.Bit3
		}
		return b | 0x80
	})
	types.RegisterHardware(types.NR11, func(v uint8) {
		if a.enabled {
			c.duty = (v & 0xC0) >> 6 // duty can only be changed when enabled
		}
		switch a.model {
		case types.CGBABC:
			if a.enabled {
				c.lengthLoad = v & 0x3F
				c.lengthCounter = 0x40 - uint(c.lengthLoad)
			}
		case types.DMGABC, types.DMG0:
			c.lengthLoad = v & 0x3F
			c.lengthCounter = 0x40 - uint(c.lengthLoad)
		default:
			// TODO add more models, for now emulate as DMG
			c.lengthLoad = v & 0x3F
			c.lengthCounter = 0x40 - uint(c.lengthLoad)
		}
	}, func() uint8 {
		if a.enabled {
			return (c.duty << 6) | 0x3F
		}
		return 0x3F
	})
	types.RegisterHardware(types.NR12, writeEnabled(a, c.setNRx2), c.getNRx2)
	types.RegisterHardware(types.NR13, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x700) | uint16(v)
	}), func() uint8 {
		return 0xFF // write only
	})
	types.RegisterHardware(types.NR14, writeEnabled(a, func(v uint8) {
		c.frequency = (c.frequency & 0x00FF) | ((uint16(v) & 0x07) << 8)
		lengthCounterEnabled := v&types.Bit6 != 0
		// obscure length counter behavior (see https://gbdev.gg8.se/wiki/articles/Gameboy_sound_hardware#Length_Counter)
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
			c.frequencyShadow = c.frequency
			if c.sweepPeriod > 0 {
				c.sweepTimer = c.sweepPeriod
			} else {
				c.sweepTimer = 8
			}
			c.sweepEnabled = c.sweepPeriod > 0 || c.shift > 0
			c.negateHasHappened = false
			if c.shift > 0 {
				c.frequencyCalculation()
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

func newVolumeChannel(channel *channel) *volumeChannel {
	return &volumeChannel{
		channel: channel,
	}
}

func (c *channel1) sweepClock() {
	if c.sweepTimer > 0 {
		c.sweepTimer--
	}
	if c.sweepTimer == 0 {
		if c.sweepPeriod > 0 {
			c.sweepTimer = c.sweepPeriod
		} else {
			c.sweepTimer = 8
		}
		if c.sweepEnabled && c.sweepPeriod > 0 {
			calculated := c.frequencyCalculation()
			if calculated <= 0x07FF && c.shift > 0 {
				c.frequencyShadow = calculated
				c.frequency = calculated
				c.frequencyCalculation()
			}
		}
	}
}

func (c *channel1) frequencyCalculation() uint16 {
	calculated := c.frequencyShadow >> c.shift
	if c.negate {
		calculated = c.frequencyShadow - 1*calculated
	}
	calculated += c.frequencyShadow
	if calculated > 0x07FF {
		c.enabled = false
	}
	c.negateHasHappened = c.negate
	return calculated
}

func (c *channel1) getAmplitude() float32 {
	if c.enabled && c.dacEnabled {
		dacInput := channel1Duty[c.duty][c.waveDutyPosition] * c.currentVolume
		dacOutput := (float32(dacInput) / 7.5) - 1
		return dacOutput
	} else {
		return 0
	}
}

var (
	channel1Duty = [4][8]uint8{
		{0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 1, 1, 1},
		{0, 1, 1, 1, 1, 1, 1, 0},
	}
)
