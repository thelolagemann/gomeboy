package apu

import (
	"fmt"
	"github.com/hajimehoshi/oto"
	"math"
	"time"
)

const (
	// sampleRate is the sample rate of the audio.
	sampleRate = 44100
	// twoPi is 2 * Pi.
	twoPi = 2 * math.Pi
	// perSample is the number of samples per second.
	perSample = 1 / float64(sampleRate)
	// cpuTicksPerSample is the number of CPU ticks per sample.
	cpuTicksPerSample = float64(4194304) / float64(sampleRate)
	// maxFrameBuffer is the maximum size of the frame buffer.
	maxFrameBuffer = 5000
)

// APU represents the GameBoy's audio processing unit. It comprises 4
// channels: 2 pulse channels, a wave channel and a noise channel. Each
// channel has is controlled by a set of registers.
//
// Channel 1 and 2 are both square channels. They can be used to play
// tones of different frequencies. Channel 3 is an arbitrary waveform
// channel that can be set in RAM. Channel 4 is a noise channel that
// can be used to play white noise.
type APU struct {
	playing bool

	memory      [52]byte
	waveformRam []byte

	player                     *oto.Player
	chan1, chan2, chan3, chan4 *Channel
	tickCounter                float64
	lVol, rVol                 float64

	audioBuffer chan [2]byte
}

// NewAPU returns a new APU.
func NewAPU() *APU {
	a := &APU{
		playing:     true,
		waveformRam: make([]byte, 0x20),
		audioBuffer: make(chan [2]byte, maxFrameBuffer),
	}

	// Initialize waveform RAM
	for i := 0x0; i < 0x20; i++ {
		if i&2 == 0 {
			a.waveformRam[i] = 0x00
		} else {
			a.waveformRam[i] = 0xFF
		}
	}

	// Initialize channels
	a.chan1 = NewChannel()
	a.chan2 = NewChannel()
	a.chan3 = NewChannel()
	a.chan4 = NewChannel()

	const bufferSeconds = 120

	// Initialize audio player
	if ctx, err := oto.NewContext(sampleRate, 2, 1, sampleRate/bufferSeconds); err != nil {
		panic(err)
	} else {
		a.player = ctx.NewPlayer()
		a.playSounds(bufferSeconds)
	}

	return a
}

// playSounds starts a goroutine that will play the audio.
func (a *APU) playSounds(bufferSeconds int) {
	frameTime := time.Second / time.Duration(bufferSeconds)
	ticker := time.NewTicker(frameTime)
	targetSamples := sampleRate / bufferSeconds

	go func() {
		var reading [2]byte
		var buffer []byte

		for range ticker.C {
			fbLen := len(a.audioBuffer)
			if fbLen >= targetSamples/2 {
				newBuffer := make([]byte, fbLen*2)
				for i := 0; i < fbLen*2; i += 2 {
					reading = <-a.audioBuffer
					newBuffer[i] = reading[0]
					newBuffer[i+1] = reading[1]
				}
				buffer = newBuffer
			}
			if a.playing {
				_, err := a.player.Write(buffer)
				if err != nil {
					panic(err)
				}
			}
		}
	}()
}

// Step advances the APU by the given number of CPU ticks and
// speed given.
func (a *APU) Step(ticks int, speed int) {
	if !a.playing {
		return
	}

	a.tickCounter += float64(ticks) / float64(speed)
	if a.tickCounter < cpuTicksPerSample {
		return
	}
	a.tickCounter -= cpuTicksPerSample

	// Sample the channels
	chn1l, chn1r := a.chan1.Sample()
	chn2l, chn2r := a.chan2.Sample()
	chn3l, chn3r := a.chan3.Sample()
	chn4l, chn4r := a.chan4.Sample()

	// Mix the channels
	valL := (chn1l + chn2l + chn3l + chn4l) / 4
	valR := (chn1r + chn2r + chn3r + chn4r) / 4

	// Send the sample to the audio buffer
	a.audioBuffer <- [2]byte{
		byte(float64(valL) * a.lVol),
		byte(float64(valR) * a.rVol),
	}
}

var soundMask = []byte{
	// 0xFF10
	0xFF, 0xC0, 0xFF, 0x00, 0x40,
	// 0xFF15
	0x00, 0xC0, 0xFF, 0x00, 0x40,
	// 0xFF1A
	0x80, 0xC0, 0x60, 0x00, 0x40,
	// 0xFF20
	0x00, 0x3F, 0xFF, 0xFF, 0x40,
	// 0xFF25
	0xFF, 0xFF, 0x80,
}

var squareLimits = map[byte]float64{
	0: -0.25,
	1: -0.5,
	2: 0,
	3: 0.5,
}

var channel3Volume = map[byte]float64{
	0: 0,
	1: 1,
	2: 0.5,
	3: 0.25,
}

// Read returns the value at the given address.
func (a *APU) Read(address uint16) uint8 {
	if address >= 0xFF30 {
		return a.waveformRam[address-0xFF30]
	}
	return a.memory[address-0xFF00] & soundMask[address-0xFF10]
}

// Write writes the value to the given address.
func (a *APU) Write(address uint16, value uint8) {
	if address < 0xFF30 {
		a.memory[address-0xFF00] = value
	}
	switch address {
	// Channel 1
	case 0xFF10:
		// Sweep period, negate, shift
		a.chan1.sweepStepLen = (a.memory[0x10] & 0b111_0000) >> 4
		a.chan1.sweepSteps = a.memory[0x10] & 0b111
		a.chan1.sweepIncrease = a.memory[0x10]&0b1000 == 0
	case 0xFF11:
		// Sound length, wave pattern duty
		duty := (value & 0b1100_0000) >> 6
		a.chan1.generator = Square(squareLimits[duty])
		a.chan1.length = int(value & 0b0011_1111)
	case 0xFF12:
		// Envelope initial volume, direction, sweep length
		envVolume, envDirection, envSweep := a.extractEnvelope(value)
		a.chan1.envVolume = int(envVolume)
		a.chan1.envSamples = int(envSweep) * sampleRate / 64
		a.chan1.envIncrease = envDirection == 1
	case 0xFF13:
		// Frequency low
		frequencyValue := uint16(a.memory[0x14]&0b111)<<8 | uint16(value)
		a.chan1.frequency = 131072 / (2048 - float64(frequencyValue))
	case 0xFF14:
		// Frequency high, initial, counter/consecutive
		frequencyValue := uint16(value&0b111)<<8 | uint16(a.memory[0x13])
		a.chan1.frequency = 131072 / (2048 - float64(frequencyValue))
		if value&0b1000_0000 != 0 {
			if a.chan1.length == 0 {
				a.chan1.length = 64
			}

			duration := -1
			if value&0b100_0000 != 0 {
				duration = int(float64(a.chan1.length)*(1/64)) * sampleRate
			}
			a.chan1.Reset(duration)
			a.chan1.envSteps = a.chan1.envVolume
			a.chan1.envStepsInit = a.chan1.envVolume
		}
	// Channel 2
	case 0xFF15:
		// unused
	case 0xFF16:
		// Sound length, wave pattern duty
		pattern := (value & 0b1100_0000) >> 6
		a.chan2.generator = Square(squareLimits[pattern])
		a.chan2.length = int(value & 0b11_1111)
	case 0xFF17:
		// Envelope initial volume, direction, sweep length
		envVolume, envDirection, envSweep := a.extractEnvelope(value)
		a.chan2.envVolume = int(envVolume)
		a.chan2.envSamples = int(envSweep) * sampleRate / 64
		a.chan2.envIncrease = envDirection == 1
	case 0xFF18:
		// Frequency low
		frequencyValue := uint16(a.memory[0x19]&0b111)<<8 | uint16(value)
		a.chan2.frequency = 131072 / (2048 - float64(frequencyValue))
	case 0xFF19:
		// Frequency high, initial, counter/consecutive
		if value&0b1000_0000 != 0 {
			if a.chan2.length == 0 {
				a.chan2.length = 64
			}

			duration := -1
			if value&0b100_0000 != 0 {
				duration = int(float64(a.chan2.length)*(1/64)) * sampleRate
			}
			a.chan2.Reset(duration)
			a.chan2.envSteps = a.chan2.envVolume
			a.chan2.envStepsInit = a.chan2.envVolume
		}
		frequencyValue := uint16(value&0b111)<<8 | uint16(a.memory[0x18])
		a.chan2.frequency = 131072 / (2048 - float64(frequencyValue))
	// Channel 3
	case 0xFF1A:
		// DAC power
		a.chan3.envStepsInit = int((value & 0b1000_0000) >> 7)
	case 0xFF1B:
		// Sound length
		a.chan3.length = int(value)
	case 0xFF1C:
		// Volume code
		selection := (value & 0b110_0000) >> 5
		a.chan3.amplitude = channel3Volume[selection]
	case 0xFF1D:
		// Frequency low
		frequencyValue := uint16(a.memory[0x1E]&0b111)<<8 | uint16(value)
		a.chan3.frequency = 65536 / (2048 - float64(frequencyValue))
	case 0xFF1E:
		// Frequency high, initial, counter/consecutive
		if value&0b1000_0000 != 0 {
			if a.chan3.length == 0 {
				a.chan3.length = 256
			}

			duration := -1
			if value&0b100_0000 != 0 {
				duration = int(256-float64(a.chan3.length)*(1/256)) * sampleRate
			}
			a.chan3.generator = Waveform(func(i int) byte { return a.waveformRam[i] })
			a.chan3.duration = duration
		}
		frequencyValue := uint16(value&0b111)<<8 | uint16(a.memory[0x1D])
		a.chan3.frequency = 65536 / (2048 - float64(frequencyValue))
	// Channel 4
	case 0xFF1F:
		// unused
	case 0xFF20:
		// Sound length
		a.chan4.length = int(value & 0b11_1111)
	case 0xFF21:
		// Envelope initial volume, direction, sweep length
		envVolume, envDirection, envSweep := a.extractEnvelope(value)
		a.chan4.envVolume = int(envVolume)
		a.chan4.envSamples = int(envSweep) * sampleRate / 64
		a.chan4.envIncrease = envDirection == 1
	case 0xFF22:
		// Polynomial counter, shift clock frequency
		shiftClock := float64((value & 0b1111_0000) >> 4)
		divRation := float64(value & 0b111)
		if divRation == 0 {
			divRation = 0.5
		}
		a.chan4.frequency = 524288 / divRation / math.Pow(2, shiftClock+1)
	case 0xFF23:
		// Counter/consecutive, initial
		if value&0x80 == 0x80 {
			duration := -1
			if value&0b100_0000 != 0 {
				duration = int(float64(61-a.chan4.length)*(1/256)) * sampleRate
			}
			a.chan4.generator = Noise()
			a.chan4.Reset(duration)
			a.chan4.envSteps = a.chan4.envVolume
			a.chan4.envStepsInit = a.chan4.envVolume
		}
	case 0xFF24:
		// Channel control / ON-OFF / Volume
		a.lVol = float64((a.memory[0x24]&0x70)>>4) / 7
		a.rVol = float64(a.memory[0x24]&0x7) / 7
	case 0xFF25:
		a.chan1.onR = value&0x1 != 0
		a.chan2.onR = value&0x2 != 0
		a.chan3.onR = value&0x4 != 0
		a.chan4.onR = value&0x8 != 0
		a.chan1.onL = value&0x10 != 0
		a.chan2.onL = value&0x20 != 0
		a.chan3.onL = value&0x40 != 0
		a.chan4.onL = value&0x80 != 0
	case 0xFF26:
		a.playing = value&0x80 != 0
	default:
		switch {
		case address >= 0xFF30 && address <= 0xFF3F:
			soundIndex := (address - 0xFF30) * 2
			a.waveformRam[soundIndex] = (value >> 4) & 0xF * 0x11
			a.waveformRam[soundIndex+1] = value & 0xF * 0x11
		default:
			panic(fmt.Sprintf("unhandled sound register write: %X", address))
		}
	}
}

// extractEnvelope extracts the envelope volume, direction and sweep
// from the given byte.
func (a *APU) extractEnvelope(value uint8) (volume, direction, sweep byte) {
	volume = (value & 0xF0) >> 4
	direction = (value & 0x8) >> 3
	sweep = value & 0x7
	return
}

// Pause pauses the APU.
func (a *APU) Pause() {
	a.playing = false
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
}
