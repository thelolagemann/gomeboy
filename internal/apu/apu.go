package apu

import (
	"encoding/binary"
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/veandco/go-sdl2/sdl"
	"math"
)

const (
	bufferSize           = 4096
	sampleRate           = 262144 // 262.144 kHz
	samplePeriod         = 4194304 / sampleRate
	frameSequencerRate   = 512
	frameSequencerPeriod = 4194304 / frameSequencerRate
)

func init() {
	// initialize SDL audio
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL audio: %v", err))
	}
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
	audioData   []byte
	chan1       *channel1
	chan2       *channel2
	chan3       *channel3
	chan4       *channel4
	TickCounter int32

	frameSequencerCounter   uint32
	frameSequencerStep      uint8
	frequencyCounter        uint32
	firstHalfOfLengthPeriod bool

	vinLeft, vinRight       bool
	volumeLeft, volumeRight uint8
	leftEnable, rightEnable [4]bool

	currentIndex uint32

	pcm12, pcm34 uint8
	waveRAM      [16]byte

	bus mmu.IOBus

	Debug struct {
		ChannelEnabled [4]bool
	}

	model         types.Model
	bufferPos     int
	audioDeviceID sdl.AudioDeviceID
	buffer        []byte

	HeldTicks uint32
}

func (a *APU) AttachBus(bus mmu.IOBus) {
	a.bus = bus
}

func (a *APU) init() {
	// Channel control (NR50 - NR52)
	types.RegisterHardware(types.NR50, func(v uint8) {
		if !a.enabled {
			return
		}
		// Channel control / ON-OFF / Volume
		a.volumeRight = v & 0x7
		a.volumeLeft = (v >> 4) & 0x7

		a.vinRight = v&types.Bit3 != 0
		a.vinLeft = v&types.Bit7 != 0
	}, func() uint8 {
		b := a.volumeRight | a.volumeLeft<<4
		if a.vinRight {
			b |= types.Bit3
		}
		if a.vinLeft {
			b |= types.Bit7
		}
		return b
	})
	types.RegisterHardware(types.NR51, func(v uint8) {
		if !a.enabled {
			return
		}
		// Channel Left/Right enable
		a.rightEnable[0] = v&types.Bit0 != 0
		a.rightEnable[1] = v&types.Bit1 != 0
		a.rightEnable[2] = v&types.Bit2 != 0
		a.rightEnable[3] = v&types.Bit3 != 0

		a.leftEnable[0] = v&types.Bit4 != 0
		a.leftEnable[1] = v&types.Bit5 != 0
		a.leftEnable[2] = v&types.Bit6 != 0
		a.leftEnable[3] = v&types.Bit7 != 0
	}, func() uint8 {
		b := uint8(0)
		for i := 0; i < 4; i++ {
			if a.rightEnable[i] {
				b |= 1 << i
			}
			if a.leftEnable[i] {
				b |= 1 << (i + 4)
			}
		}
		return b
	})
	types.RegisterHardware(types.NR52, func(v uint8) {
		if v&types.Bit7 == 0 && a.enabled {
			for i := types.NR10; i <= types.NR51; i++ {
				a.bus.Write(i, 0)
			}
			a.enabled = false
		} else if v&types.Bit7 != 0 && !a.enabled {
			// Power on
			a.enabled = true
			a.frameSequencerStep = 0

		}
	}, func() uint8 {
		b := uint8(0)
		if a.enabled {
			b |= types.Bit7
		}

		if a.chan1.channel.enabled {
			b |= types.Bit0
		}
		if a.chan2.channel.enabled {
			b |= types.Bit1
		}
		if a.chan3.channel.enabled {
			b |= types.Bit2
		}
		if a.chan4.channel.enabled {
			b |= types.Bit3
		}

		return b | 0x70
	})
}

func (a *APU) SetModel(model types.Model) {
	a.model = model
}

// NewAPU returns a new APU.
func NewAPU() *APU {
	a := &APU{
		playing:               false,
		frequencyCounter:      16,
		frameSequencerCounter: 8192,
		frameSequencerStep:    0,
		buffer:                make([]byte, bufferSize),
	}
	a.init()

	// Initialize channels
	a.chan1 = newChannel1(a)
	a.chan2 = newChannel2(a)
	a.chan3 = newChannel3(a)
	a.chan4 = newChannel4(a)

	// initialize audio
	spec := &sdl.AudioSpec{
		Freq:     sampleRate,
		Format:   sdl.AUDIO_F32SYS,
		Channels: 2,
		Samples:  bufferSize,
		Callback: nil,
		UserData: nil,
	}

	if id, err := sdl.OpenAudioDevice("", false, spec, nil, 0); err != nil {
		panic(err)
	} else {
		a.audioDeviceID = id
	}

	sdl.PauseAudioDevice(a.audioDeviceID, false)

	return a
}

// Tick
func (a *APU) TickM() {
	// we don't need to do anything if the APU is disabled, not playing or no held ticks

	for i := uint8(0); i < 4; i++ {
		if a.frameSequencerCounter--; a.frameSequencerCounter <= 0 {
			a.frameSequencerCounter = frameSequencerPeriod
			a.firstHalfOfLengthPeriod = a.frameSequencerStep&types.Bit0 == 0

			switch a.frameSequencerStep {
			case 0:
				a.chan1.lengthStep()
				a.chan2.lengthStep()
				a.chan3.lengthStep()
				a.chan4.lengthStep()
			case 2:
				a.chan1.lengthStep()
				a.chan2.lengthStep()
				a.chan3.lengthStep()
				a.chan4.lengthStep()
				a.chan1.sweepClock()
			case 4:
				a.chan1.lengthStep()
				a.chan2.lengthStep()
				a.chan3.lengthStep()
				a.chan4.lengthStep()
			case 6:
				a.chan1.lengthStep()
				a.chan2.lengthStep()
				a.chan3.lengthStep()
				a.chan4.lengthStep()
				a.chan1.sweepClock()
			case 7:
				a.chan1.volumeStep()
				a.chan2.volumeStep()
				a.chan4.volumeStep()
			}

			a.frameSequencerStep = (a.frameSequencerStep + 1) & 7
		}

		a.chan1.step()
		a.chan2.step()
		a.chan3.step()
		a.chan4.step()

		if a.frequencyCounter--; a.frequencyCounter <= 0 {
			a.frequencyCounter = samplePeriod

			channel1Amplitude := a.chan1.getAmplitude()
			channel2Amplitude := a.chan2.getAmplitude()
			channel3Amplitude := a.chan3.getAmplitude()
			channel4Amplitude := a.chan4.getAmplitude()

			left := float32(0)
			right := float32(0)
			for i, amplitude := range []float32{channel1Amplitude, channel2Amplitude, channel3Amplitude, channel4Amplitude} {
				if a.leftEnable[i] && !a.Debug.ChannelEnabled[i] {
					left += amplitude
				}
				if a.rightEnable[i] && !a.Debug.ChannelEnabled[i] {
					right += amplitude
				}
			}

			left = ((float32(a.volumeLeft) / 7) * left) / 4
			right = ((float32(a.volumeRight) / 7) * right) / 4

			var buf [8]byte
			binary.LittleEndian.PutUint32(buf[:4], math.Float32bits(left))
			binary.LittleEndian.PutUint32(buf[4:], math.Float32bits(right))

			// push to internal buffer
			copy(a.buffer[a.bufferPos:], buf[:])
			a.bufferPos += 8

			// push to SDL buffer when internal buffer is full
			if a.bufferPos >= bufferSize {
				// wait until the buffer is empty
				if err := sdl.QueueAudio(a.audioDeviceID, a.buffer); err != nil {
					panic(err)
				}
				a.bufferPos = 0
			}
		}
	}

}

// Read returns the value at the given address.
func (a *APU) Read(address uint16) uint8 {
	if address >= 0xFF30 && address <= 0xFF3F {
		return a.chan3.readWaveRAM(address)
	}
	panic(fmt.Sprintf("unhandled APU read at address: 0x%04X", address))
}

// Write writes the value to the given address.
func (a *APU) Write(address uint16, value uint8) {

	if address >= 0xFF30 && address <= 0xFF3F {
		a.chan3.writeWaveRAM(address, value)
		return
	}
	panic("invalid address")
}

// Pause pauses the APU.
func (a *APU) Pause() {
	a.playing = false
	a.enabled = false
	sdl.PauseAudioDevice(a.audioDeviceID, true)
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
	a.enabled = true
	// sdl.PauseAudioDevice(a.audioDeviceID, false)
}
