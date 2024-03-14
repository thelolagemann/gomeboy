package apu

import (
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
	"unsafe"
)

const (
	bufferSize           = 2048
	emulatedSampleRate   = 96000
	samplePeriod         = 4194304 / emulatedSampleRate
	frameSequencerRate   = 512
	frameSequencerPeriod = 4194304 / frameSequencerRate
)

var (
	chargeFactor = 0.998943
	capacitors   [4]float32
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

	pcm12, pcm34 uint8

	Debug struct {
		ChannelEnabled [4]bool
	}

	model     types.Model
	bufferPos int
	buffer    []uint8

	lastUpdate uint64
	s          *scheduler.Scheduler
	b          *io.Bus

	playBack func([]uint8)
}

func highPass(channel int, in float32, dacEnabled bool) float32 {
	var out float32
	if dacEnabled {
		out = (in) - capacitors[channel]
		capacitors[channel] = (in - out) * float32(chargeFactor)
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
		buffer:             make([]uint8, bufferSize),
		s:                  s,
		b:                  b,
		Samples:            make(Samples, emulatedSampleRate/64),
	}
	b.ReserveAddress(types.NR50, func(v byte) byte {
		if b.IsBooting() {
			a.volumeRight = v & 0x7
			a.volumeLeft = v >> 4 & 7

			a.vinRight = v&types.Bit3 != 0
			a.vinLeft = v&types.Bit7 != 0
		}
		if !a.enabled {
			return b.Get(types.NR50)
		}

		a.volumeRight = v & 0x7
		a.volumeLeft = (v >> 4) & 0x7

		a.vinRight = v&types.Bit3 != 0
		a.vinLeft = v&types.Bit7 != 0

		return v
	})
	b.ReserveAddress(types.NR51, func(v byte) byte {
		if b.IsBooting() {
			a.rightEnable[0] = v&types.Bit0 != 0
			a.rightEnable[1] = v&types.Bit1 != 0
			a.rightEnable[2] = v&types.Bit2 != 0
			a.rightEnable[3] = v&types.Bit3 != 0

			a.leftEnable[0] = v&types.Bit4 != 0
			a.leftEnable[1] = v&types.Bit5 != 0
			a.leftEnable[2] = v&types.Bit6 != 0
			a.leftEnable[3] = v&types.Bit7 != 0
			return v
		}
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
	})
	b.ReserveAddress(types.NR52, func(v byte) byte {
		if b.IsBooting() {
			a.enabled = v&types.Bit7 != 0
			a.chan1.enabled = v&types.Bit0 != 0
			a.chan2.enabled = v&types.Bit1 != 0
			a.chan3.enabled = v&types.Bit2 != 0
			a.chan4.enabled = v&types.Bit3 != 0
			return v
		}
		if v&types.Bit7 == 0 && a.enabled {
			for i := types.NR10; i <= types.NR51; i++ {
				b.Write(i, 0)
			}
			a.enabled = false
		} else if v&types.Bit7 != 0 && !a.enabled {
			// power on
			a.enabled = true
			a.frameSequencerStep = 0
			a.chan3.volumeCodeShift = 4

			// if the DIV APU bit is set when powering on,
			// the first DIV event is skipped
			// https://github.com/LIJI32/SameSuite/blob/master/apu/div_write_trigger_10.asm
			if a.s.SysClock()&a.s.DivAPUBit() != 0 {
				a.frameSequencerStep = 1
			}
		}

		return b.LazyRead(types.NR52) | 0x70
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
	b.RegisterGBCHandler(func() {
		b.ReserveAddress(types.PCM12, func(v byte) byte {
			if a.model == types.CGBABC || a.model == types.CGB0 {
				return channel1Duty[a.chan1.duty][a.chan1.waveDutyPosition]*a.chan1.currentVolume |
					channel2Duty[a.chan2.duty][a.chan2.waveDutyPosition]*a.chan2.currentVolume&0xf<<4
			}
			return 0xFF
		})
		b.ReserveLazyReader(types.PCM12, func() byte {
			return channel1Duty[a.chan1.duty][a.chan1.waveDutyPosition]*a.chan1.currentVolume |
				channel2Duty[a.chan2.duty][a.chan2.waveDutyPosition]*a.chan2.currentVolume&0xf<<4
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
	// grab amplitudes from channels
	channel1Amplitude := a.chan1.getAmplitude()
	channel2Amplitude := a.chan2.getAmplitude()
	channel3Amplitude := a.chan3.getAmplitude()
	channel4Amplitude := a.chan4.getAmplitude()

	// mix amplitudes in "mixer"
	left := uint16(0)
	right := uint16(0)
	for i, amplitude := range []uint8{channel1Amplitude, channel2Amplitude, channel3Amplitude, channel4Amplitude} {
		if a.leftEnable[i] {
			left += uint16(amplitude)
		}
		if a.rightEnable[i] {
			right += uint16(amplitude)
		}
	}
	left *= uint16(a.volumeLeft) << 7
	right *= uint16(a.volumeRight) << 7

	*(*uint32)(unsafe.Pointer(&a.buffer[a.bufferPos])) = uint32(left)<<16 | uint32(right)

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

	a.s.ScheduleEvent(scheduler.APUSample, samplePeriod)
}

// Pause pauses the APU.
func (a *APU) Pause() {
	a.playing = false
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
}

func (a *APU) AttachPlayback(playback func([]uint8)) {
	// attach provided channel to listeners
	a.playBack = playback
}
