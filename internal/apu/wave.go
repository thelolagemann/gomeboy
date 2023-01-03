package apu

import (
	"math"
	"math/rand"
)

// WaveGenerator is the interface for generating waveforms.
type WaveGenerator func(t float64) byte

// Square is a square wave generator.
func Square(mod float64) WaveGenerator {
	return func(t float64) byte {
		if math.Sin(t) <= mod {
			return 0xFF
		}
		return 0x00
	}
}

// Waveform is a waveform generator. This is used by channel 3.
func Waveform(ram func(i int) byte) WaveGenerator {
	return func(t float64) byte {
		idx := int(math.Floor(t/twoPi*32)) % 0x20
		return ram(idx)
	}
}

// Noise is a noise generator. This is used by channel 4.
func Noise() WaveGenerator {
	var last float64
	var val byte
	return func(t float64) byte {
		if t-last > twoPi {
			last = t
			val = byte(rand.Intn(2)) * 0xFF
		}
		return val
	}
}
