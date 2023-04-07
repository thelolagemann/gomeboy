package apu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/scheduler"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/veandco/go-sdl2/sdl"
	"unsafe"
)

const (
	bufferSize           = 2048
	emulatedSampleRate   = 4194304 / 16
	samplePeriod         = 4194304 / emulatedSampleRate
	frameSequencerRate   = 512
	frameSequencerPeriod = 4194304 / frameSequencerRate
)

var (
	audioDeviceID sdl.AudioDeviceID
)

func init() {
	// initialize SDL audio
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		panic(fmt.Sprintf("failed to initialize SDL audio: %v", err))
	}

	// open audio device
	var err error
	if audioDeviceID, err = sdl.OpenAudioDevice("", false, &sdl.AudioSpec{
		Freq:     emulatedSampleRate,
		Format:   sdl.AUDIO_U16SYS,
		Channels: 2,
		Samples:  bufferSize,
	}, nil, 0); err != nil {
		panic(fmt.Sprintf("failed to open audio device: %v", err))
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

	model     types.Model
	bufferPos int
	buffer    []byte

	HeldTicks uint32

	s *scheduler.Scheduler
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
	}, registerSetter(func(v interface{}) {
		a.volumeRight = v.(uint8) & 0x7
		a.volumeLeft = (v.(uint8) >> 4) & 0x7

		a.vinRight = v.(uint8)&types.Bit3 != 0
		a.vinLeft = v.(uint8)&types.Bit7 != 0
	}))
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
		if !a.enabled {
			return 0
		}
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
	}, registerSetter(func(v interface{}) {
		a.rightEnable[0] = v.(uint8)&types.Bit0 != 0
		a.rightEnable[1] = v.(uint8)&types.Bit1 != 0
		a.rightEnable[2] = v.(uint8)&types.Bit2 != 0
		a.rightEnable[3] = v.(uint8)&types.Bit3 != 0

		a.leftEnable[0] = v.(uint8)&types.Bit4 != 0
		a.leftEnable[1] = v.(uint8)&types.Bit5 != 0
		a.leftEnable[2] = v.(uint8)&types.Bit6 != 0
		a.leftEnable[3] = v.(uint8)&types.Bit7 != 0
	}))
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
	}, registerSetter(func(v interface{}) {
		// if you are reading this, you may be wondering why this setter
		// forces the channels on despite writes to NR52 just setting
		// the enabled flag. this is because writes to NR52 do not actually
		// control the channel enable flag, but rather the power state of the
		// APU. the setter here provides a way for the APU to be automatically
		// configured with a provided state, which is useful for both boot ROM
		// and state loading.
		a.enabled = v.(uint8)&types.Bit7 != 0
		a.chan1.enabled = v.(uint8)&types.Bit0 != 0
		a.chan2.enabled = v.(uint8)&types.Bit1 != 0
		a.chan3.enabled = v.(uint8)&types.Bit2 != 0
		a.chan4.enabled = v.(uint8)&types.Bit3 != 0
	}))

	types.RegisterHardware(types.PCM12, func(v uint8) {
		// PCM12 is a read-only register that returns the current PCM12 value
	}, func() uint8 {
		if a.model == types.CGBABC || a.model == types.CGB0 {
			return a.pcm12
		}
		return 0xFF
	})
	types.RegisterHardware(types.PCM34, func(v uint8) {
		// PCM34 is a read-only register that returns the current PCM34 value
	}, func() uint8 {
		if a.model == types.CGBABC || a.model == types.CGB0 {
			return a.pcm34
		}
		return 0xFF
	})
}

func (a *APU) SetModel(model types.Model) {
	a.model = model
}

// NewAPU returns a new APU.
func NewAPU(s *scheduler.Scheduler) *APU {
	a := &APU{
		playing:               false,
		frequencyCounter:      16,
		frameSequencerCounter: 8192,
		frameSequencerStep:    0,
		buffer:                make([]byte, bufferSize),
		s:                     s,
	}
	a.init()

	// Initialize channels
	a.chan1 = newChannel1(a)
	a.chan2 = newChannel2(a)
	a.chan3 = newChannel3(a)
	a.chan4 = newChannel4(a)

	// initialize audio

	s.RegisterEvent(scheduler.APUFrameSequencer, a.stepFrameSequencer)
	s.RegisterEvent(scheduler.APUSample, a.sample)

	a.stepFrameSequencer()
	a.sample()
	sdl.PauseAudioDevice(audioDeviceID, false)
	return a
}

func (a *APU) stepFrameSequencer() {
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

	// schedule next frame sequencer step in 8192 cycles
	a.s.ScheduleEvent(scheduler.APUFrameSequencer, frameSequencerPeriod)
}

func (a *APU) sample() {
	channel1Amplitude := a.chan1.getAmplitude()
	channel2Amplitude := a.chan2.getAmplitude()
	channel3Amplitude := a.chan3.getAmplitude()
	channel4Amplitude := a.chan4.getAmplitude()

	left := uint16(0)
	right := uint16(0)
	for i, amplitude := range []uint8{channel1Amplitude, channel2Amplitude, channel3Amplitude, channel4Amplitude} {
		if a.leftEnable[i] && !a.Debug.ChannelEnabled[i] {
			left += uint16(amplitude)
		}
		if a.rightEnable[i] && !a.Debug.ChannelEnabled[i] {
			right += uint16(amplitude)
		}
	}

	// push to internal buffer using unsafe
	*(*uint16)(unsafe.Pointer(&a.buffer[a.bufferPos])) = left * 128 * uint16(a.volumeLeft)
	*(*uint16)(unsafe.Pointer(&a.buffer[a.bufferPos+2])) = right * 128 * uint16(a.volumeRight)

	a.bufferPos += 4

	// push to SDL buffer when internal buffer is full
	if a.bufferPos >= bufferSize {
		// are we playing?
		if a.playing {
			// wait until the buffer is empty
			if err := sdl.QueueAudio(audioDeviceID, a.buffer); err != nil {
				panic(err)
			}
		}
		a.bufferPos = 0
	}

	// schedule next sample in samplePeriod cycles
	a.s.ScheduleEvent(scheduler.APUSample, samplePeriod)
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
	sdl.PauseAudioDevice(audioDeviceID, true)
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
	a.enabled = true
	// sdl.PauseAudioDevice(a.audioDeviceID, false)
}
