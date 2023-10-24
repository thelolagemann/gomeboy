package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"math"
)

const (
	bufferSize           = 256
	emulatedSampleRate   = 65536
	samplePeriod         = 4194304 / emulatedSampleRate
	frameSequencerRate   = 512
	frameSequencerPeriod = 4194304 / frameSequencerRate
)

var (
	chargeFactor = math.Pow(0.998943, 4194304/emulatedSampleRate)
)

type Sample struct {
	Channel1, Channel2, Channel3, Channel4 uint8
}

type Samples []Sample

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
	Samples          Samples

	audioData []byte
	chan1     *channel1
	chan2     *channel2
	chan3     *channel3
	chan4     *channel4

	frameSequencerStep      uint8
	firstHalfOfLengthPeriod bool

	vinLeft, vinRight       bool
	volumeLeft, volumeRight uint8
	leftEnable, rightEnable [4]bool
	capacitors              [4]float64

	pcm12, pcm34 uint8

	Debug struct {
		ChannelEnabled [4]bool
	}

	model     types.Model
	bufferPos int
	buffer    []byte

	lastUpdate uint64
	s          *scheduler.Scheduler
	b          *io.Bus

	playBack func([]byte)
}

func (a *APU) highPass(channel int, in uint8, dacEnabled bool) uint8 {
	var out uint8
	if dacEnabled {
		out = uint8(float64(in) - a.capacitors[channel])
		a.capacitors[channel] = float64(in-out) * chargeFactor
	}

	return out
}

func (a *APU) SetModel(model types.Model) {
	a.model = model
}

// NewAPU returns a new APU.
func NewAPU(s *scheduler.Scheduler, b *io.Bus) *APU {
	a := &APU{
		playing:            false,
		enabled:            true,
		frameSequencerStep: 0,
		buffer:             make([]byte, bufferSize),
		s:                  s,
		b:                  b,
		Samples:            make(Samples, emulatedSampleRate/64),
	}
	b.ReserveAddress(types.NR50, func(v byte) byte {
		if !a.enabled {
			return b.Get(types.NR50)
		}

		a.volumeRight = v & 0x7
		a.volumeLeft = (v >> 4) & 0x7

		a.vinRight = v&types.Bit3 != 0
		a.vinLeft = v&types.Bit7 != 0

		return v

		// TODO onset
	})
	b.ReserveSetAddress(types.NR50, func(v any) {
		a.volumeRight = v.(uint8) & 0x7
		a.volumeLeft = (v.(uint8) >> 4) & 0x7

		a.vinRight = v.(uint8)&types.Bit3 != 0
		a.vinLeft = v.(uint8)&types.Bit7 != 0

		b.Set(types.NR50, v.(byte))
	})
	b.ReserveAddress(types.NR51, func(v byte) byte {
		if !a.enabled {
			return b.Get(types.NR51)
		}

		a.rightEnable[0] = v&types.Bit0 != 0
		a.rightEnable[1] = v&types.Bit1 != 0
		a.rightEnable[2] = v&types.Bit2 != 0
		a.rightEnable[3] = v&types.Bit3 != 0

		a.leftEnable[0] = v&types.Bit4 != 0
		a.leftEnable[1] = v&types.Bit5 != 0
		a.leftEnable[2] = v&types.Bit6 != 0
		a.leftEnable[3] = v&types.Bit7 != 0

		return v

		// TODO onset
	})
	b.ReserveSetAddress(types.NR51, func(val any) {
		v := val.(uint8)
		a.rightEnable[0] = v&types.Bit0 != 0
		a.rightEnable[1] = v&types.Bit1 != 0
		a.rightEnable[2] = v&types.Bit2 != 0
		a.rightEnable[3] = v&types.Bit3 != 0

		a.leftEnable[0] = v&types.Bit4 != 0
		a.leftEnable[1] = v&types.Bit5 != 0
		a.leftEnable[2] = v&types.Bit6 != 0
		a.leftEnable[3] = v&types.Bit7 != 0

		b.Set(types.NR51, v)

	})
	b.ReserveAddress(types.NR52, func(v byte) byte {
		if v&types.Bit7 == 0 && a.enabled {
			aChans := []*channel{a.chan1.channel, a.chan2.channel, a.chan3.channel, a.chan4.channel}

			oldVals := [4]bool{}
			for i, ch := range aChans {
				oldVals[i] = ch.enabled
			}
			for i := types.NR10; i <= types.NR51; i++ {
				b.Write(i, 0)
			}
			a.enabled = false
		} else if v&types.Bit7 != 0 && !a.enabled {
			// power on
			a.enabled = true
			a.frameSequencerStep = 0
		}

		//fmt.Printf("NR52: %08b %08b\n", b.Get(types.NR52)|0x70, v)
		return b.LazyRead(types.NR52) | 0x70

		// TODO onset
	})
	b.ReserveSetAddress(types.NR52, func(val any) {
		v := val.(uint8)
		a.enabled = v&types.Bit7 != 0
		a.chan1.enabled = v&types.Bit0 != 0
		a.chan2.enabled = v&types.Bit1 != 0
		a.chan3.enabled = v&types.Bit2 != 0
		a.chan4.enabled = v&types.Bit3 != 0

		//b.Set(types.NR52, v|0x70)
	})
	b.ReserveLazyReader(types.NR52, func() byte {
		b := uint8(0x70)
		if a.enabled {
			b |= types.Bit7
		}

		aChans := []*channel{a.chan1.channel, a.chan2.channel, a.chan3.channel, a.chan4.channel}
		for i, ch := range aChans {
			if ch.enabled {
				b |= uint8(1 << i)
			}
		}

		return b
	})

	for i := uint16(0xff30); i <= 0xff3f; i++ {
		cI := i
		b.ReserveLazyReader(cI, func() byte {
			return a.chan3.readWaveRAM(cI)
		})
	}
	b.WhenGBC(func() {

		b.ReserveAddress(types.PCM12, func(v byte) byte {
			if a.model == types.CGBABC || a.model == types.CGB0 {
				return a.pcm12
			}
			return 0xFF
		})
		b.ReserveAddress(types.PCM34, func(v byte) byte {
			if a.model == types.CGBABC || a.model == types.CGB0 {
				return a.pcm34
			}
			return 0xFF
		})
	})

	for i := 0xFF30; i < 0xFF40; i++ {
		cI := i
		b.ReserveAddress(uint16(cI), func(v byte) byte {
			a.chan3.writeWaveRAM(uint16(cI), v)

			return v
		})
	}

	// Initialize channels
	a.chan1 = newChannel1(a, b)
	a.chan2 = newChannel2(a, b)
	a.chan3 = newChannel3(a, b)
	a.chan4 = newChannel4(a, b)

	// initialize audio

	s.RegisterEvent(scheduler.APUFrameSequencer, a.scheduledFrameSequencer)
	s.RegisterEvent(scheduler.APUSample, a.sample)

	s.ScheduleEvent(scheduler.APUFrameSequencer, 0)
	a.sample()
	s.ScheduleEvent(scheduler.APUChannel3, 0)

	// nrx3 regs always read 0xFF
	b.Set(types.NR13, 0xFF)
	b.Set(types.NR23, 0xFF)
	b.Set(types.NR33, 0xFF)
	b.Set(types.NR43, 0xFF)

	return a
}

func (a *APU) StepFrameSequencer() {

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

func (a *APU) scheduledFrameSequencer() {
	a.StepFrameSequencer()

	// schedule next frame sequencer step in 8192 cycles
	a.s.ScheduleEvent(scheduler.APUFrameSequencer, frameSequencerPeriod)
}

func (a *APU) sample() {
	channel1Amplitude := a.chan1.getAmplitude()
	channel2Amplitude := a.chan2.getAmplitude()
	channel3Amplitude := a.chan3.getAmplitude()
	channel4Amplitude := a.chan4.getAmplitude()

	// apply high-pass filter
	//channel1Amplitude = a.highPass(0, channel1Amplitude, a.chan1.channel.dacEnabled)
	//channel2Amplitude = a.highPass(1, channel2Amplitude, a.chan2.channel.dacEnabled)
	//channel3Amplitude = a.highPass(2, channel3Amplitude, a.chan3.channel.dacEnabled)
	//channel4Amplitude = a.highPass(3, channel4Amplitude, a.chan4.channel.dacEnabled)

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

	left *= 128 * uint16(a.volumeLeft)
	right *= 128 * uint16(a.volumeRight)

	a.buffer[a.bufferPos] = uint8(left)
	a.buffer[a.bufferPos+1] = uint8(left >> 8)
	a.buffer[a.bufferPos+2] = uint8(right)
	a.buffer[a.bufferPos+3] = uint8(right >> 8)

	a.bufferPos += 4

	// push to broadcast when internal buffer is full
	if a.bufferPos >= bufferSize {
		// are we playing?
		if a.playing && a.playBack != nil {
			// broadcast buffer to listeners
			a.playBack(a.buffer)
		}
		a.bufferPos = 0
	}
}

// Pause pauses the APU.
func (a *APU) Pause() {
	a.playing = false
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
}

func (a *APU) AttachPlayback(playback func([]byte)) {
	// attach provided channel to listeners
	a.playBack = playback
}
