package apu

import (
	"github.com/thelolagemann/gomeboy/internal/types"
)

type channel struct {
	enabled    bool
	dacEnabled bool

	// NRx1
	lengthCounter uint

	// NRx4
	frequencyTimer       uint64
	lengthCounterEnabled bool
}

type volumeChannel struct {
	*channel

	// NRx1
	duty       uint8
	lengthLoad uint8

	// NRx2
	startingVolume  uint8
	envelopeAddMode bool
	period          uint8

	// NRx3/4
	frequency uint16

	waveDutyPosition         uint8
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

func (v *volumeChannel) setDuty(duty uint8) {
	v.duty = (duty & 0xC0) >> 6
}

func (v *volumeChannel) setLength(length uint8) {
	v.lengthLoad = length & 0x3F
	v.lengthCounter = 0x40 - uint(v.lengthLoad)
}

func (v *volumeChannel) setNRx1(v1 uint8) {
	v.lengthLoad = v1 & 0x3F
	v.lengthCounter = 0x40 - uint(v.lengthLoad)
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
