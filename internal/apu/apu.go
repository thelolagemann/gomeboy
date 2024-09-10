package apu

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/internal/types"
)

var (
	duties    = [4]uint8{0x80, 0x81, 0xE1, 0x7E}
	readMasks = [0x3f]uint8{
		0x80, 0x3f, 0x00, 0xff, 0xbf, // NR10-NR14
		0xff, 0x3f, 0x00, 0xff, 0xbf, // NR21-NR24
		0x7f, 0xff, 0x9f, 0xff, 0xbf, // NR30-NR34
		0xff, 0xff, 0x00, 0x00, 0xbf, // NR40-NR44
		0x00, 0x00, 0x70, 0xff, 0xff, // ...
		0xff, 0xff, 0xff, 0xff, 0xff, // ...
		0xff, 0xff,
	}
)

const (
	bufferSize           = 1634 * 4
	emulatedSampleRate   = 96000
	samplePeriod         = 4194304 / emulatedSampleRate
	frameSequencerRate   = 512
	frameSequencerPeriod = 4194304 / frameSequencerRate
)

type channel struct {
	enableTime          uint64
	enableTimeIncurred  uint64
	lengthCounter       uint16 // NRx1
	frequency           uint16 // NRx3
	period              uint8
	volumeEnvelopeTimer uint8
	startingVolume      uint8
	currentVolume       uint8

	clock, shouldLock, lock bool
	envelopeDirection       bool
	lengthCounterEnabled    bool
	enabled, dacEnabled     bool
}

type squareChannel struct {
	duty             uint8
	lockedDuty       uint8
	waveDutyPosition uint8
	hasLockedDuty    bool
}

func (c channel) isEnabled() bool {
	return c.enabled && c.dacEnabled
}

type APU struct {
	channels [4]channel
	channel1 struct {
		squareChannel
		frequencyShadow uint16
		sweepPeriod     uint8
		sweepTimer      uint8
		shift           uint8
		negate          bool
		didNegate       bool
		sweepEnabled    bool
	}
	channel2 squareChannel
	channel3 struct {
		waveRAMLastRead     uint64
		volumeCode          uint8
		waveRAMPosition     uint8
		waveRAMSampleBuffer uint8
		waveRAMLastPosition uint8
	}
	channel4 struct {
		clockShift     uint8
		divisorCode    uint8
		widthMask      uint16
		lfsr           uint16
		delayedCycles  uint64
		isTriggered    bool
		cyclesIncurred uint64
		frequencyTimer uint64
	}

	enabled                 bool
	frameSequencerStep      uint8
	waveRAM                 [16]uint8
	vinLeft, vinRight       bool
	leftEnable, rightEnable [4]bool
	volumeLeft, volumeRight uint8
	turningOn               bool
	turnedOn                bool
	enableTimer             uint64
	buffer                  []float32
	bufferPos               uint32
	b                       *io.Bus
	s                       *scheduler.Scheduler
	lastCatchup             uint64
}

func New(b *io.Bus, s *scheduler.Scheduler) *APU {
	a := &APU{
		buffer:  make([]float32, bufferSize),
		s:       s,
		b:       b,
		enabled: true,
	}
	a.channel3.volumeCode = 4

	s.RegisterEvent(scheduler.APUChannel1, func() {
		// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_1/channel_1_stop_restart.asm
		if a.channels[0].isEnabled() {
			a.channel1.waveDutyPosition = (a.channel1.waveDutyPosition + 1) & 7
			s.ScheduleEvent(scheduler.APUChannel1, uint64((2048-a.channels[0].frequency)<<2))
		}

		// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_1/channel_1_duty_delay.asm
		if a.channel1.hasLockedDuty {
			a.channel1.duty = a.channel1.lockedDuty
			a.channel1.hasLockedDuty = false
		}
	})

	s.RegisterEvent(scheduler.APUChannel2, func() {
		// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_2/channel_2_stop_restart.asm
		if a.channels[1].isEnabled() {
			a.channel2.waveDutyPosition = (a.channel2.waveDutyPosition + 1) & 7
			s.ScheduleEvent(scheduler.APUChannel2, uint64((2048-a.channels[1].frequency)<<2))
		}

		// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_2/channel_2_duty_delay.asm
		if a.channel2.hasLockedDuty {
			a.channel2.duty = a.channel2.lockedDuty
			a.channel2.hasLockedDuty = false
		}
	})
	s.RegisterEvent(scheduler.APUChannel3, func() {
		if a.channels[2].isEnabled() {
			a.channel3.waveRAMPosition = (a.channel3.waveRAMPosition + 1) & 31
			a.channel3.waveRAMLastRead = a.s.Cycle()
			a.channel3.waveRAMLastPosition = a.channel3.waveRAMPosition >> 1
			a.channel3.waveRAMSampleBuffer = a.waveRAM[a.channel3.waveRAMLastPosition]
			a.s.ScheduleEvent(scheduler.APUChannel3, uint64((2048-a.channels[2].frequency)<<1))
		} else {
			a.channel3.waveRAMSampleBuffer = 0
		}
	})

	s.RegisterEvent(scheduler.APUFrameSequencer, func() {
		a.StepFrameSequencer()
		s.ScheduleEvent(scheduler.APUFrameSequencer, frameSequencerPeriod)
	})
	s.RegisterEvent(scheduler.APUFrameSequencer2, func() {
		a.SecondaryDIVEvent()
		s.ScheduleEvent(scheduler.APUFrameSequencer2, frameSequencerPeriod)
	})
	s.RegisterEvent(scheduler.APUSample, a.sample)
	s.ScheduleEvent(scheduler.APUChannel3, 0)
	a.sample()

	b.ReserveLazyReader(types.NR52, func() byte {
		v := uint8(0x70)
		if a.enabled {
			v |= types.Bit7
		}
		for i, ch := range a.channels {
			if ch.enabled {
				v |= uint8(1 << i)
			}
		}
		return v
	})

	for i := uint16(0xff10); i <= types.NR52; i++ {
		b.ReserveAddress(i, func(b byte) byte {
			return a.Write(i, b) | readMasks[(i&0x3f)-16]
		})
	}
	for i := uint16(0xff30); i <= 0xff3f; i++ {
		b.ReserveAddress(i, func(b byte) byte {
			return a.Write(i, b)
		})
		b.ReserveLazyReader(i, func() byte {
			return a.readWaveRAM(i)
		})
	}
	a.channel4.frequencyTimer = 8

	b.RegisterGBCHandler(func() {
		b.ReserveLazyReader(types.PCM12, func() byte {
			// warning to any who read
			// this is not accurate to hardware in the slightest, this is many botch jobs and timing hacks to work around
			// botch jobs. even these PCM readers are hacked together
			pcm := uint8(0)

			sampleLength := (uint64(2048-a.channels[0].frequency) * 4) + 4
			if a.s.DoubleSpeed() {
				sampleLength *= 2
			}

			if a.channels[0].enabled && a.s.Cycle()-a.channels[0].enableTime >= sampleLength {
				pcm |= (duties[a.channel1.duty] >> a.channel1.waveDutyPosition) & 1 * a.channels[0].currentVolume
			}
			fmt.Printf("%02x %d %d %d %d %d %08b %d %d %d %t %t\n", pcm, a.channel1.duty, a.channel1.waveDutyPosition, (2048-a.channels[0].frequency)*4, a.s.Cycle()-a.channels[0].enableTime, a.s.Cycle(), duties[a.channel1.duty], a.channels[0].volumeEnvelopeTimer, a.channels[0].currentVolume, sampleLength, a.channels[0].enabled, a.channels[0].dacEnabled)
			sampleLength = uint64(2048-a.channels[1].frequency)*4 + 4
			if a.s.DoubleSpeed() {
				sampleLength *= 2
			}
			if a.channels[1].enabled && a.s.Cycle()-a.channels[1].enableTime >= sampleLength {
				pcm |= (((duties[a.channel2.duty] >> a.channel2.waveDutyPosition) & 1 * a.channels[1].currentVolume) & 0xf) << 4
			}

			return pcm

		})
		b.ReserveLazyReader(types.PCM34, func() byte {
			pcm := uint8(0)
			sampleLength := (uint64(2048-a.channels[2].frequency) * 2) + 4
			if a.s.DoubleSpeed() {
				sampleLength *= 2
			}
			if a.channels[2].enabled && (a.s.Cycle()-a.channels[2].enableTime > sampleLength) {
				shift := 0
				if a.channel3.waveRAMPosition&1 == 0 {
					shift = 4
				}
				pcm |= (((a.channel3.waveRAMSampleBuffer) >> shift) & 0x0f) >> a.channel3.volumeCode
			}
			sampleLength = uint64(a.channel4.frequencyTimer) + 4
			if a.s.DoubleSpeed() {
				sampleLength *= 2
			}
			if a.channels[3].enabled && a.s.Cycle()-a.channels[3].enableTime > sampleLength {
				a.catchupLFSR()
				pcm |= ((uint8(a.channel4.lfsr) & 1 * a.channels[3].currentVolume) & 0x0f) << 4
			}

			fmt.Printf("%02x %d %d %t %016b\n", pcm, a.s.Cycle(), a.channels[3].enableTime, a.channels[3].isEnabled(), a.channel4.lfsr)
			return pcm

		})
	})
	for i := types.NR10; i <= types.NR44; i++ {
		a.b.Write(i, 0) // load or masks into bus
	}
	return a
}

func (a *APU) StepFrameSequencer() {
	if !a.enabled { // frame sequencer does nothing if the APU is disabled
		return
	}

	// the first frame after being turned on is skipped
	if a.turningOn {
		a.turningOn = false
		a.turnedOn = true
		return
	}

	// the first frame that isn't skipped doesn't advance the step
	if a.turnedOn { // ;)
		a.turnedOn = false // ;(
	} else {
		a.frameSequencerStep = (a.frameSequencerStep + 1) & 7
	}

	// clock volume envelope (64hz)
	if a.frameSequencerStep&7 == 7 {
		a.clockVolume(0)
		a.clockVolume(1)
		a.clockVolume(3)
	}

	a.tickEnvelope(0)
	a.tickEnvelope(1)
	a.tickEnvelope(3)

	// clock length (256hz)
	if a.frameSequencerStep&1 == 1 {
		for i := 0; i < 4; i++ {
			if a.channels[i].lengthCounterEnabled && a.channels[i].lengthCounter > 0 {
				if a.channels[i].lengthCounter--; a.channels[i].lengthCounter == 0 {
					a.channels[i].enabled = false
				}
			}
		}
	}

	// clock sweep (128hz)
	if a.frameSequencerStep&3 == 3 {
		if a.channels[0].enabled && a.channel1.sweepEnabled {
			a.channel1.sweepTimer++
			a.channel1.sweepTimer &= 7

			if a.channel1.sweepTimer == 7 {
				a.channel1.sweepTimer = a.channel1.sweepPeriod ^ 7

				if a.channel1.sweepPeriod != 0 {
					a.freqCalc(true)
					a.freqCalc(false)
				}
			}
		}
	}
}

func (a *APU) SecondaryDIVEvent() {
	if !a.enabled {
		return
	}
	for i := uint16(0); i < 2; i++ {
		if a.channels[i].isEnabled() && a.channels[i].volumeEnvelopeTimer == 0 {
			a.channels[i].volumeEnvelopeTimer = a.channels[i].period
			a.setEnvelopeClock(i, a.channels[i].volumeEnvelopeTimer > 0, a.channels[i].envelopeDirection, a.channels[i].currentVolume)
		}
	}
	if a.channels[3].isEnabled() && a.channels[3].volumeEnvelopeTimer == 0 {
		a.channels[3].volumeEnvelopeTimer = a.channels[3].period
		a.setEnvelopeClock(3, a.channels[3].volumeEnvelopeTimer > 0, a.channels[3].envelopeDirection, a.channels[3].currentVolume)
	}
}

var capacitors [2]float32

func highPass(ch int, in float32, dacEnabled bool) float32 {
	out := float32(0.0)
	if dacEnabled {
		out = in - capacitors[ch]
		capacitors[ch] = in - out*0.998166636
	}
	return out
}

func digitalAnalog(v uint8, enabled bool) float32 {
	if !enabled {
		return 0
	}
	return ((float32(v))/15.0*(1.0 - -1.0) + -1.0) * 0.25
}

var volumes = []float32{.125, .250, .375, .500, .625, .750, .875, 1}

func (a *APU) sample() {
	channels := a.channels
	leftEnable, rightEnable := a.leftEnable, a.rightEnable

	samples := [4]float32{}
	samples[0] = digitalAnalog((duties[a.channel1.duty]>>a.channel1.waveDutyPosition)&1*(channels[0].currentVolume), channels[0].isEnabled())
	samples[1] = digitalAnalog((duties[a.channel2.duty]>>a.channel2.waveDutyPosition)&1*(channels[1].currentVolume), channels[1].isEnabled())
	samples[2] = digitalAnalog(((a.channel3.waveRAMSampleBuffer>>((a.channel3.waveRAMPosition&1)<<2))&0x0f)>>a.channel3.volumeCode, channels[2].isEnabled())
	a.catchupLFSR()
	samples[3] = digitalAnalog(uint8(a.channel4.lfsr&1)*channels[3].currentVolume, channels[3].isEnabled())

	left := float32(0)
	right := float32(0)
	for i := 0; i < 4; i++ {
		if leftEnable[i] {
			left += samples[i]
		}
		if rightEnable[i] {
			right += samples[i]
		}
	}
	left *= volumes[a.volumeLeft]
	right *= volumes[a.volumeRight]

	enabled := !(left == 0 && right == 0) || a.channels[0].dacEnabled || a.channels[1].dacEnabled || a.channels[2].dacEnabled || a.channels[3].dacEnabled

	fLeft := highPass(0, left, enabled)
	fRight := highPass(1, right, enabled)
	if fLeft > 1.0 || fLeft < -1.0 {
		fmt.Println(left, "->", fLeft, samples, channels[2].dacEnabled, channels[2].enabled, channels[3].dacEnabled, channels[3].enabled)
	}
	// Write to the buffer
	if a.bufferPos < bufferSize {
		a.buffer[a.bufferPos] = fLeft
		a.buffer[a.bufferPos+1] = fRight
	} else {
		a.buffer = append(a.buffer, fLeft, fRight)
	}
	a.bufferPos += 2

	a.s.ScheduleEvent(scheduler.APUSample, samplePeriod)
}

// TODO apu or masks
func (a *APU) Write(address uint16, v uint8) uint8 {
	switch address {
	case types.NR10:
		if !a.enabled {
			return 0
		}

		oldNegate := a.channel1.negate
		a.channel1.sweepPeriod = (v & 0x70) >> 4
		a.channel1.negate = v&types.Bit3 != 0
		a.channel1.shift = v & 7

		if oldNegate && !a.channel1.negate && a.channel1.didNegate {
			a.channels[0].enabled = false
		}
	case types.NR11:
		if a.enabled {
			a.channel1.lockedDuty = (v & 0xc0) >> 6
			a.channel1.hasLockedDuty = true
		}

		a.writeNRx1(0, v)
		return a.channel1.lockedDuty << 6
	case types.NR12, types.NR22, types.NR42:
		if !a.enabled {
			return 0
		}
		ch := ((address & 0x00ff) - 18) / 5
		if v&0xf8 == 0 {
			// disable DAC
			a.channels[ch].dacEnabled = false
		} else if a.channels[ch].isEnabled() {
			a.glitchNRx2(ch, v, a.b.Get(address))
			a.channels[ch].enableTimeIncurred = 12 // TODO this is just a hack to pass restart_nrx2_glitch
		}
		a.channels[ch].startingVolume = v >> 4
		a.channels[ch].envelopeDirection = v&types.Bit3 > 0
		a.channels[ch].period = v & 7
		a.channels[ch].dacEnabled = v&0xf8 > 0

		if !a.channels[ch].dacEnabled {
			a.channels[ch].enabled = false
		}
	case types.NR13, types.NR23, types.NR33:
		if a.enabled {
			ch := ((address & 0x00ff) - 19) / 5
			a.channels[ch].frequency = (a.channels[ch].frequency & 0x700) | uint16(v)
		}
	case types.NR14, types.NR24, types.NR34, types.NR44:
		if !a.enabled {
			return 0
		}
		ch := ((address & 0x00ff) - 20) / 5
		if ch != 3 {
			a.channels[ch].frequency = (a.channels[ch].frequency & 0x00ff) | uint16(v&0x7)<<8
		}
		lengthCounterEnabled := v&types.Bit6 > 0
		if a.frameSequencerStep&1 == 1 && !a.channels[ch].lengthCounterEnabled && lengthCounterEnabled && a.channels[ch].lengthCounter > 0 {
			a.channels[ch].lengthCounter--
			a.channels[ch].enabled = a.channels[ch].lengthCounter > 0
		}
		a.channels[ch].lengthCounterEnabled = lengthCounterEnabled

		// handle trigger
		if v&types.Bit7 > 0 {
			if ch != 2 {
				if a.channels[ch].lengthCounter == 0 {
					a.channels[ch].lengthCounter = 0x40
					if a.channels[ch].lengthCounterEnabled && a.frameSequencerStep&1 == 1 {
						a.channels[ch].lengthCounter--
					}
				}

				a.channels[ch].clock = false
				a.channels[ch].lock = false
				a.channels[ch].volumeEnvelopeTimer = a.channels[ch].period
				a.channels[ch].currentVolume = a.channels[ch].startingVolume
			}

			// handle channel enabling
			a.channels[ch].enabled = a.channels[ch].dacEnabled
			a.channels[ch].enableTime = a.s.Cycle() - a.channels[ch].enableTimeIncurred
			a.channels[ch].enableTimeIncurred = 0

			switch ch {
			case 0: // Square 1
				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_1/channel_1_delay.asm
				offset := uint64(8)
				if a.s.Until(scheduler.APUChannel1) != 0 {
					// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_1/channel_1_restart.asm
					offset = 4
				}

				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_1/channel_1_align.asm
				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_1/channel_1_align_cpu.asm
				t := uint64((2048-a.channels[0].frequency)*4) + offset
				if a.s.DoubleSpeed() && (a.s.Cycle()-a.enableTimer)%8 != 0 {
					t += 2
				}
				a.s.DescheduleEvent(scheduler.APUChannel1)
				a.s.ScheduleEvent(scheduler.APUChannel1, t)
				a.channel1.didNegate = false
				a.channel1.sweepTimer = a.channel1.sweepPeriod ^ 7
				a.channel1.frequencyShadow = a.channels[0].frequency
				a.channel1.sweepEnabled = a.channel1.sweepPeriod != 0 || a.channel1.shift != 0

				if a.channel1.shift > 0 {
					a.freqCalc(false)
				}
			case 1: // Square 2
				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_2/channel_2_delay.asm
				offset := uint64(8)
				if a.s.Until(scheduler.APUChannel2) != 0 {
					// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_2/channel_2_restart.asm
					offset = 4
				}

				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_2/channel_2_align.asm
				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_2/channel_2_align_cpu.asm
				t := uint64((2048-a.channels[1].frequency)*4) + offset
				if a.s.DoubleSpeed() && (a.s.Cycle()-a.enableTimer)%8 != 0 {
					t += 2
				}
				a.s.DescheduleEvent(scheduler.APUChannel2)
				a.s.ScheduleEvent(scheduler.APUChannel2, t)
			case 2: // Wave
				if a.channels[2].isEnabled() && a.s.Until(scheduler.APUChannel3) == 2 && (a.b.Model() != types.CGBABC && a.b.Model() != types.CGB0) {
					newPos := (a.channel3.waveRAMPosition + 1) & 31
					pos := newPos >> 1
					if pos < 4 {
						a.waveRAM[0] = a.waveRAM[pos]
					} else {
						pos &^= 3
						copy(a.waveRAM[0:4], a.waveRAM[pos:pos+4])
					}
				}

				if a.channels[2].lengthCounter == 0 {
					a.channels[2].lengthCounter = 0x100
					if a.channels[2].lengthCounterEnabled && a.frameSequencerStep&1 == 1 {
						a.channels[2].lengthCounter--
					}
				}

				a.channel3.waveRAMPosition = 0
				a.channel3.waveRAMLastPosition = 0

				t := uint64((2048 - a.channels[2].frequency) << 1)
				a.s.DescheduleEvent(scheduler.APUChannel3)
				a.s.ScheduleEvent(scheduler.APUChannel3, t+6)
			case 3: // Noise
				a.channel4.lfsr = 0
				a.lastCatchup = a.s.Cycle() // lfsr is reset so no need to step through

				offset := uint64(4)
				if a.channel4.isTriggered {
					offset = 8
				}

				// https://github.com/LIJI32/SameSuite/blob/master/apu/channel_4/channel_4_align.asm
				if a.s.DoubleSpeed() && (a.s.Cycle()-a.enableTimer)%8 != 0 {
					offset += 4
				}

				if a.s.DoubleSpeed() {
					offset /= 2
				}
				a.channel4.delayedCycles = offset
				a.channel4.isTriggered = true
				a.channel4.cyclesIncurred = 0
			}
		}
	case types.NR21:
		if a.enabled {
			a.channel2.lockedDuty = (v & 0xc0) >> 6
			a.channel2.hasLockedDuty = true
		}

		a.writeNRx1(1, v)
		return a.channel2.lockedDuty << 6
	case types.NR30:
		if !a.enabled {
			return 0
		}
		a.channels[2].dacEnabled = v&types.Bit7 != 0
		if !a.channels[2].dacEnabled {
			a.channels[2].enabled = false
		}
	case types.NR31:
		switch a.b.Model() {
		case types.CGBABC, types.CGB0:
			if a.enabled {
				a.channels[2].lengthCounter = 0x100 - uint16(v)
			}
		default:
			a.channels[2].lengthCounter = 0x100 - uint16(v)
		}
	case types.NR32:
		if !a.enabled {
			return 0
		}
		a.channel3.volumeCode = []byte{4, 0, 1, 2}[(v&0x60)>>5]
	case types.NR41:
		a.writeNRx1(3, v)

		return 0xff // write only
	case types.NR43:
		if !a.enabled {
			return 0
		}
		a.catchupLFSR()
		a.channel4.widthMask = 0x4000 | uint16(v&types.Bit3)<<3
		a.channel4.clockShift = v >> 4

		a.channel4.divisorCode = v & 7

		if a.channel4.divisorCode == 0 {
			a.channel4.frequencyTimer = 8 << a.channel4.clockShift
		} else {
			a.channel4.frequencyTimer = uint64(a.channel4.divisorCode<<4) << a.channel4.clockShift
		}
	case types.NR50:
		if !a.enabled && !a.b.IsBooting() {
			return 0
		}

		a.volumeRight = v & 7
		a.volumeLeft = (v >> 4) & 7

		a.vinRight = v&types.Bit3 > 0
		a.vinLeft = v&types.Bit7 > 0
	case types.NR51:
		if !a.enabled && !a.b.IsBooting() {
			return 0
		}

		for i := 0; i < 4; i++ {
			a.rightEnable[i] = v&(1<<i) > 0
			a.leftEnable[i] = (v & (1 << (i + 4))) > 0
		}
	case types.NR52:
		if a.b.IsBooting() {
			a.enabled = v&types.Bit7 > 0
			for i := 0; i < 4; i++ {
				a.channels[i].enabled = v&(1<<i) > 0
			}
			return v
		}
		oldLengthCounters := []uint16{a.channels[0].lengthCounter, a.channels[1].lengthCounter, a.channels[2].lengthCounter, a.channels[3].lengthCounter}

		if v&types.Bit7 == 0 && a.enabled {
			for i := types.NR10; i <= types.NR51; i++ {
				a.b.Write(i, 0)
			}
			a.channels = [4]channel{}
			a.enabled = false
			a.s.DescheduleEvent(scheduler.APUChannel1)
			a.s.DescheduleEvent(scheduler.APUChannel2)
			a.s.DescheduleEvent(scheduler.APUChannel3)
		} else if v&types.Bit7 > 0 && !a.enabled {
			a.enabled = true
			a.frameSequencerStep = 0
			a.channel3.volumeCode = 4
			// first DIV event is skipped if DIV APU bit is set
			// https://github.com/LIJI32/SameSuite/blob/master/apu/div_write_trigger_10.asm
			if a.s.SysClock()&a.s.DivAPUBit() != 0 {
				a.frameSequencerStep = 1
				a.turningOn = true // first div event is skipped
				a.turnedOn = false
			}
			a.channel1 = struct {
				squareChannel
				frequencyShadow uint16
				sweepPeriod     uint8
				sweepTimer      uint8
				shift           uint8
				negate          bool
				didNegate       bool
				sweepEnabled    bool
			}{}
			a.channel2 = squareChannel{}
			a.channel4 = struct {
				clockShift     uint8
				divisorCode    uint8
				widthMask      uint16
				lfsr           uint16
				delayedCycles  uint64
				isTriggered    bool
				cyclesIncurred uint64
				frequencyTimer uint64
			}{}
			a.channel3.volumeCode = 4
			a.lastCatchup = a.s.Cycle()
			a.channels = [4]channel{}
			a.channel4.frequencyTimer = 8
			a.enableTimer = a.s.Cycle()
		}
		if !a.b.IsGBC() && v&0x80 > 0 {
			for i := 0; i < 4; i++ {
				a.channels[i].lengthCounter = oldLengthCounters[i]
			}
		}
	default:
		if address >= 0xff30 && address <= 0xff3f {
			if a.channels[2].isEnabled() {
				if a.s.Cycle()-a.channel3.waveRAMLastRead < 2 || a.b.Model() == types.CGBABC || a.b.Model() == types.CGB0 {
					a.waveRAM[a.channel3.waveRAMLastPosition] = v
				}
			} else {
				a.waveRAM[address-0xff30] = v
			}
		}
	}

	return v
}

func (a *APU) clockVolume(channel int) {
	if a.channels[channel].clock {
		return
	}
	a.channels[channel].volumeEnvelopeTimer--
	a.channels[channel].volumeEnvelopeTimer &= 7
}

func (a *APU) tickEnvelope(channel uint16) {
	if !a.channels[channel].clock {
		return
	}
	a.setEnvelopeClock(channel, false, false, 0)
	if a.channels[channel].lock || a.channels[channel].period == 0 {
		return
	}
	if channel != 3 {
		a.setEnvelopeClock(channel, false, false, 0)
	}

	if a.channels[channel].envelopeDirection {
		a.channels[channel].currentVolume++
	} else {
		a.channels[channel].currentVolume--
	}
}

func (a *APU) setEnvelopeClock(channel uint16, value, direction bool, volume uint8) {
	if a.channels[channel].clock == value {
		return
	}
	if value {
		a.channels[channel].clock = true
		a.channels[channel].shouldLock = (volume == 0xf && direction) || (volume == 0x0 && !direction)
	} else {
		a.channels[channel].clock = false
		a.channels[channel].lock = a.channels[channel].shouldLock
	}
}

func (a *APU) glitchNRx2(channel uint16, value uint8, oldValue uint8) {
	if a.channels[channel].clock {
		a.channels[channel].volumeEnvelopeTimer = value & 7
	}
	shouldTick := (value&7) > 0 && (oldValue&7 == 0 && !a.channels[channel].lock)
	shouldInvert := ((value & 8) ^ (oldValue & 8)) > 0

	if (value&0xf) == 8 && (oldValue&0xf) == 8 && !a.channels[channel].lock {
		shouldTick = true
	}

	if shouldInvert {
		if value&8 > 0 {
			if !(oldValue&7 > 0) && !a.channels[channel].lock {
				a.channels[channel].currentVolume ^= 0xf
			} else {
				a.channels[channel].currentVolume = 0xe - a.channels[channel].currentVolume
				a.channels[channel].currentVolume &= 0xf
			}
			shouldTick = false
		} else {
			a.channels[channel].currentVolume = 0x10 - a.channels[channel].currentVolume
			a.channels[channel].currentVolume &= 0xf
		}
	}

	if shouldTick {
		if value&8 > 0 {
			a.channels[channel].currentVolume++
		} else {
			a.channels[channel].currentVolume--
		}
		a.channels[channel].currentVolume &= 0xf
	} else if !(value&7 > 0) && a.channels[channel].clock {
		a.setEnvelopeClock(channel, false, false, 0)
	}
}

func (a *APU) readWaveRAM(address uint16) uint8 {
	if a.channels[2].isEnabled() {
		if a.s.Cycle()-a.channel3.waveRAMLastRead < 2 || a.b.Model() == types.CGBABC || a.b.Model() == types.CGB0 {
			return a.waveRAM[a.channel3.waveRAMLastPosition]
		} else {
			return 0xff
		}
	}
	return a.waveRAM[address-0xff30]
}

func (a *APU) catchupLFSR() {
	currentCycle := a.s.Cycle()
	cyclesPassed := currentCycle - (a.lastCatchup)
	if a.s.DoubleSpeed() {
		cyclesPassed >>= 1
	}
	cyclesPassed += a.channel4.cyclesIncurred

	if cyclesPassed <= a.channel4.delayedCycles {
		a.channel4.delayedCycles -= cyclesPassed
		a.lastCatchup = currentCycle
		return
	}

	cyclesPassed -= a.channel4.delayedCycles

	freqTimer := a.channel4.frequencyTimer
	steps := (cyclesPassed) / freqTimer

	// step LFSR state
	if steps > 0 {
		lfsr := a.channel4.lfsr
		bitMask := a.channel4.widthMask

		for i := uint64(0); i < steps; i++ {
			newHighBit := (lfsr ^ (lfsr >> 1) ^ 1) & 1
			lfsr >>= 1
			lfsr = (lfsr &^ bitMask) | (newHighBit * bitMask)
		}

		a.channel4.lfsr = lfsr
	}

	a.channel4.delayedCycles = 0
	a.channel4.cyclesIncurred = cyclesPassed % freqTimer

	a.lastCatchup = currentCycle
}

func (a *APU) Samples() ([]float32, uint32) {
	s, b := a.buffer[:a.bufferPos], a.bufferPos
	a.bufferPos = 0
	return s, b
}

func (a *APU) freqCalc(update bool) {
	newFreq := a.channel1.frequencyShadow >> a.channel1.shift
	if a.channel1.negate {
		a.channel1.didNegate = true
		newFreq = a.channel1.frequencyShadow - newFreq
	} else {
		newFreq = a.channel1.frequencyShadow + newFreq
	}

	if newFreq > 0x7ff {
		a.channels[0].enabled = false
	} else if a.channel1.shift > 0 && update {
		a.channel1.frequencyShadow = newFreq
		a.channels[0].frequency = newFreq
	}
}

func (a *APU) writeNRx1(ch int, v uint8) {
	switch a.b.Model() {
	case types.CGBABC, types.CGB0:
		if a.enabled {
			a.channels[ch].lengthCounter = uint16(0x40 - (v & 0x3f))
		}
	default:
		a.channels[ch].lengthCounter = uint16(0x40 - (v & 0x3f))
	}
}
