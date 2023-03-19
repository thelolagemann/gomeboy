package apu

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"math"
	"sync"
)

const (
	// sampleRate is the sample rate of the audio.
	sampleRate = 48000
	// twoPi is 2 * Pi.
	twoPi = 2 * math.Pi
	// perSample is the number of samples per second.
	perSample = 1 / float64(sampleRate)

	// cpuTicksPerSample is the number of CPU ticks per sample.
	cpuTicksPerSample = (4194304) / (sampleRate)
)

var (
	context *audio.Context
)

func init() {
	context = audio.NewContext(sampleRate)

	// wait for context to be ready
}

// APU represents the GameBoy's audio processing unit. It comprises 4
// channels: 2 pulse channels, a wave channel and a noise channel. Each
// channel has is controlled by a set of addresses.
//
// Channel 1 and 2 are both square channels. They can be used to play
// tones of different frequencies. Channel 3 is an arbitrary waveform
// channel that can be set in RAM. Channel 4 is a noise channel that
// can be used to play white noise.
type APU struct {
	playing, enabled bool

	memory      [52]byte
	waveformRam []byte

	chan1, chan2, chan3, chan4 *Channel
	TickCounter                int32
	lVol, rVol                 float64

	audioBuffer  *buffer
	player       *audio.Player
	currentIndex uint32
}

type buffer struct {
	data           []byte
	readPosition   int
	writePosition  int
	size           int
	bytesToCollect int
	sync.RWMutex

	sampleChan chan [2]uint16
}

func (b *buffer) start() {
	go func() {
		for {
			select {
			case sample := <-b.sampleChan:
				b.Lock()
				// copy sample to buffer
				copy(b.data[b.writePosition:b.writePosition+4], []byte{byte(sample[0]), byte(sample[0] >> 8), byte(sample[1]), byte(sample[1] >> 8)})

				b.writePosition = (b.writePosition + 4) % b.size
				b.bytesToCollect += 4
				b.Unlock()
			}
		}
	}()
}

func (b *buffer) Read(p []byte) (int, error) {
	b.RLock()
	defer b.RUnlock()
	var bytesCollected int
	for i := 0; i < b.bytesToCollect && i < len(p); i += 4 {
		p[i] = b.data[b.readPosition]
		p[i+1] = b.data[b.readPosition+1]
		p[i+2] = b.data[b.readPosition+2]
		p[i+3] = b.data[b.readPosition+3]
		b.readPosition = (b.readPosition + 4) % b.size
		bytesCollected += 4
	}

	b.bytesToCollect -= bytesCollected
	return bytesCollected, nil
}

var orMasks = []byte{
	// NR1x
	0x80, 0x3F, 0x00, 0xFF, 0xBF,
	// NR2x
	0xFF, 0x3F, 0x00, 0xFF, 0xBF,
	// NR3x
	0x7F, 0xFF, 0x9F, 0xFF, 0xBF,
	// NR4x
	0xFF, 0xFF, 0x00, 0x00, 0xBF,
	// NR5x
	0x00, 0x00, 0x70,
	// FF27 - FF2F 0xFF
}

func (a *APU) registerHardware(address uint16, w func(value uint8)) {
	types.RegisterHardware(
		address,
		func(v uint8) {
			a.Tick()
			a.memory[address-0xFF00] = v
			w(v)
		},
		func() uint8 {
			return a.memory[address-0xFF00] | orMasks[address-0xFF10]
		},
	)
}

func (a *APU) init() {
	// setup addresses

	// Channel 1 (NR10 - NR14)
	a.registerHardware(types.NR10, func(v uint8) {
		// Sweep period, negate, shift
		a.chan1.sweepStepLen = (a.memory[0x10] & 0b111_0000) >> 4
		a.chan1.sweepSteps = a.memory[0x10] & 0b111
		a.chan1.sweepIncrease = a.memory[0x10]&0b1000 == 0
	})
	a.registerHardware(types.NR11, func(v uint8) {
		// Sound length, wave pattern duty
		duty := (v & 0b1100_0000) >> 6
		a.chan1.generator = Square(squareLimits[duty])
		a.chan1.length = int(v & 0b0011_1111)
	})
	a.registerHardware(types.NR12, func(v uint8) {
		// Envelope initial volume, direction, sweep length
		envVolume, envDirection, envSweep := a.extractEnvelope(v)
		a.chan1.envVolume = int(envVolume)
		a.chan1.envSamples = int(envSweep) * sampleRate / 64
		a.chan1.envIncrease = envDirection == 1
	})
	a.registerHardware(types.NR13, func(v uint8) {
		// Frequency low
		frequencyValue := uint16(a.memory[0x14]&0b111)<<8 | uint16(v)
		a.chan1.frequency = 131072 / (2048 - float64(frequencyValue))
	})
	a.registerHardware(types.NR14, func(v uint8) {
		// Frequency high, initial, counter/consecutive
		frequencyValue := uint16(v&0b111)<<8 | uint16(a.memory[0x13])
		a.chan1.frequency = 131072 / (2048 - float64(frequencyValue))
		if v&0b1000_0000 != 0 {
			if a.chan1.length == 0 {
				a.chan1.length = 64
			}

			duration := -1
			if v&0b100_0000 != 0 {
				duration = int(float64(a.chan1.length)*(1/64)) * sampleRate
			}
			a.chan1.Reset(duration)
			a.chan1.envSteps = a.chan1.envVolume
			a.chan1.envStepsInit = a.chan1.envVolume
		}
	})
	a.registerHardware(0xFF15, types.NoWrite)

	// Channel 2 (NR20 - NR24)
	a.registerHardware(types.NR21, func(v uint8) {
		// Sound length, wave pattern duty
		duty := (v & 0b1100_0000) >> 6
		a.chan2.generator = Square(squareLimits[duty])
		a.chan2.length = int(v & 0b11_1111)
	})
	a.registerHardware(types.NR22, func(v uint8) {
		// Envelope initial volume, direction, sweep length
		envVolume, envDirection, envSweep := a.extractEnvelope(v)
		a.chan2.envVolume = int(envVolume)
		a.chan2.envSamples = int(envSweep) * sampleRate / 64
		a.chan2.envIncrease = envDirection == 1
	})
	a.registerHardware(types.NR23, func(v uint8) {
		// Frequency low
		frequencyValue := uint16(a.memory[0x19]&0b111)<<8 | uint16(v)
		a.chan2.frequency = 131072 / (2048 - float64(frequencyValue))
	})
	a.registerHardware(types.NR24, func(v uint8) {
		// Frequency high, initial, counter/consecutive
		if v&0b1000_0000 != 0 {
			if a.chan2.length == 0 {
				a.chan2.length = 64
			}

			duration := -1
			if v&0b100_0000 != 0 {
				duration = int(float64(a.chan2.length)*(1/64)) * sampleRate
			}
			a.chan2.Reset(duration)
			a.chan2.envSteps = a.chan2.envVolume
			a.chan2.envStepsInit = a.chan2.envVolume
		}
		frequencyValue := uint16(v&0b111)<<8 | uint16(a.memory[0x18])
		a.chan2.frequency = 131072 / (2048 - float64(frequencyValue))
	})

	// Channel 3 (NR30 - NR34)
	a.registerHardware(types.NR30, func(v uint8) {
		// DAC power
		a.chan3.envStepsInit = int((v & 0b1000_0000) >> 7)
	})
	a.registerHardware(types.NR31, func(v uint8) {
		// Sound length
		a.chan3.length = int(v)
	})
	a.registerHardware(types.NR32, func(v uint8) {
		selection := (v & 0b110_0000) >> 5
		a.chan3.amplitude = channel3Volume[selection]
	})
	a.registerHardware(types.NR33, func(v uint8) {
		// Frequency low
		frequencyValue := uint16(a.memory[0x1E]&0b111)<<8 | uint16(v)
		a.chan3.frequency = 65536 / (2048 - float64(frequencyValue))
	})
	a.registerHardware(types.NR34, func(v uint8) {
		// Frequency high, initial, counter/consecutive
		if v&0b1000_0000 != 0 {
			if a.chan3.length == 0 {
				a.chan3.length = 256
			}

			duration := -1
			if v&0b100_0000 != 0 {
				duration = int(256-float64(a.chan3.length)*(1/256)) * sampleRate
			}
			a.chan3.generator = Waveform(func(i int) byte { return a.waveformRam[i] })
			a.chan3.duration = duration
		}
		frequencyValue := uint16(v&0b111)<<8 | uint16(a.memory[0x1D])
		a.chan3.frequency = 65536 / (2048 - float64(frequencyValue))
	})

	// Channel 4 (NR40 - NR44)
	a.registerHardware(types.NR41, func(v uint8) {
		// Sound length
		a.chan4.length = int(v & 0b11_1111)
	})
	a.registerHardware(types.NR42, func(v uint8) {
		// Envelope initial volume, direction, sweep length
		envVolume, envDirection, envSweep := a.extractEnvelope(v)
		a.chan4.envVolume = int(envVolume)
		a.chan4.envSamples = int(envSweep) * sampleRate / 64
		a.chan4.envIncrease = envDirection == 1
	})
	a.registerHardware(types.NR43, func(v uint8) {
		// Polynomial counter, shift clock frequency
		shiftClock := float64((v & 0b1111_0000) >> 4)
		divRation := float64(v & 0b111)
		if divRation == 0 {
			divRation = 0.5
		}
		a.chan4.frequency = 524288 / divRation / math.Pow(2, shiftClock+1)
	})
	a.registerHardware(types.NR44, func(v uint8) {
		// Counter/consecutive, initial
		if v&0x80 == 0x80 {
			duration := -1
			if v&0b100_0000 != 0 {
				duration = int(float64(61-a.chan4.length)*(1/256)) * sampleRate
			}
			a.chan4.generator = Noise()
			a.chan4.Reset(duration)
			a.chan4.envSteps = a.chan4.envVolume
			a.chan4.envStepsInit = a.chan4.envVolume
		}
	})

	// Channel control (NR50 - NR52)
	a.registerHardware(types.NR50, func(v uint8) {
		// Channel control / ON-OFF / Volume
		a.lVol = float64((a.memory[0x24]&0x70)>>4) / 7
		a.rVol = float64(a.memory[0x24]&0x7) / 7
	})
	a.registerHardware(types.NR51, func(v uint8) {
		a.chan1.onR = v&0x1 != 0
		a.chan2.onR = v&0x2 != 0
		a.chan3.onR = v&0x4 != 0
		a.chan4.onR = v&0x8 != 0
		a.chan1.onL = v&0x10 != 0
		a.chan2.onL = v&0x20 != 0
		a.chan3.onL = v&0x40 != 0
		a.chan4.onL = v&0x80 != 0
	})
	a.registerHardware(types.NR52, func(v uint8) {
		// Sound on/off
		a.playing = v&0x80 != 0
	})
}

// NewAPU returns a new APU.
func NewAPU() *APU {
	b := &buffer{data: make([]byte, sampleRate*10), size: sampleRate * 10, sampleChan: make(chan [2]uint16, sampleRate)}
	a := &APU{
		playing:     false,
		waveformRam: make([]byte, 0x20),
		audioBuffer: b,
	}
	a.init()

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

	// initialize audio
	player, err := context.NewPlayer(a.audioBuffer)
	if err != nil {
		panic(fmt.Sprintf("failed to create player: %v", err))
	}
	a.player = player
	a.player.SetBufferSize(100)
	return a
}

// Tick advances the APU by the given number of CPU ticks and
// speed given.
func (a *APU) Tick() {
	if !a.playing || !a.enabled {
		return
	}
	if a.TickCounter < cpuTicksPerSample {
		return
	}

	for i := int32(0); i < a.TickCounter/cpuTicksPerSample; i++ {
		// sample channels
		chn1l, chn1r := a.chan1.Sample()
		chn2l, chn2r := a.chan2.Sample()
		chn3l, chn3r := a.chan3.Sample()
		chn4l, chn4r := a.chan4.Sample()

		// mix channels
		valL := uint16((chn1l+chn2l+chn3l+chn4l)/4) * 128
		valR := uint16((chn1r+chn2r+chn3r+chn4r)/4) * 128

		// write to buffer
		a.audioBuffer.sampleChan <- [2]uint16{valL, valR}
		a.TickCounter -= cpuTicksPerSample
	}
}

var squareLimits = []float64{
	0: -0.25,
	1: -0.5,
	2: 0,
	3: 0.5,
}

var channel3Volume = []float64{
	0: 0,
	1: 1,
	2: 0.5,
	3: 0.25,
}

// Read returns the value at the given address.
func (a *APU) Read(address uint16) uint8 {
	if address >= 0xFF30 && address <= 0xFF3F {
		return a.waveformRam[address-0xFF30]
	}
	panic(fmt.Sprintf("unhandled APU read at address: 0x%04X", address))
}

// Write writes the value to the given address.
func (a *APU) Write(address uint16, value uint8) {
	if address >= 0xFF30 && address <= 0xFF3F {
		a.Tick()
		soundIndex := (address - 0xFF30) * 2
		a.waveformRam[soundIndex] = (value >> 4) & 0xF * 0x11
		a.waveformRam[soundIndex+1] = value & 0xF * 0x11
		return
	}
	panic("invalid address")
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
	a.enabled = false
	a.player.Pause()
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
	a.enabled = true
	a.audioBuffer.start()
	a.player.Play()
}
