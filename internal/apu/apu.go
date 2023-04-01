package apu

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/thelolagemann/go-gameboy/internal/mmu"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"math"
	"sync"
)

const (
	// twoPi is 2 * Pi.
	twoPi = 2 * math.Pi
	// perSample is the number of samples per second.
	perSample = 1 / float64(sampleRate)

	// cpuTicksPerSample is the number of CPU ticks per sample.
	cpuTicksPerSample = (4194304) / (sampleRate)
)

const (
	bufferSize           = 1024
	sampleRate           = 65536
	samplePeriod         = 4194304 / sampleRate
	frameSequencerRate   = 512
	frameSequencerPeriod = 4194304 / frameSequencerRate
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
	audioData   []byte
	chan1       *channel1
	chan2       *channel2
	chan3       *channel3
	chan4       *channel4
	TickCounter int32

	frameSequencerCounter   uint
	frameSequencerStep      uint
	frequencyCounter        uint
	firstHalfOfLengthPeriod bool

	vinLeft, vinRight       bool
	volumeLeft, volumeRight uint8
	leftEnable, rightEnable [4]bool
	volumes                 [4]float64

	audioBuffer  *buffer
	player       *audio.Player
	currentIndex uint32

	pcm12, pcm34 uint8
	waveRAM      [16]byte

	bus mmu.IOBus
}

func (a *APU) AttachBus(bus mmu.IOBus) {
	a.bus = bus
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
			// Power off (registers are reset)
			for i := types.NR10; i <= types.NR51; i++ {
				if i == types.NR41 {
					continue // Power off does not reset length counter
				}
				a.bus.Write(i, 0)
			}
			a.enabled = false
		} else if v&types.Bit7 != 0 && !a.enabled {
			// Power on
			a.enabled = true
			a.frameSequencerStep = 0
			a.chan1.lengthCounter = 0
			a.chan2.lengthCounter = 0
			a.chan3.lengthCounter = 0
			a.chan4.lengthCounter = 0
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

// NewAPU returns a new APU.
func NewAPU() *APU {
	b := &buffer{data: make([]byte, sampleRate*10), size: sampleRate * 10, sampleChan: make(chan [2]uint16, sampleRate)}
	a := &APU{
		playing:               false,
		audioBuffer:           b,
		volumes:               [4]float64{1.0, 1.0, 1.0, 1.0},
		frequencyCounter:      95,
		frameSequencerCounter: 8192,
		frameSequencerStep:    0,
	}
	a.init()

	// Initialize channels
	a.chan1 = newChannel1(a)
	a.chan2 = newChannel2(a)
	a.chan3 = newChannel3(a)
	a.chan4 = newChannel4(a)

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
	if a.frameSequencerCounter--; a.frameSequencerCounter <= 0 {
		a.frameSequencerCounter = frameSequencerPeriod

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

		left, right := 0, 0
		output := uint8(0)

		// get the output from each channel
		for i := 0; i < 4; i++ {
			switch i {
			case 0:
				output = a.chan1.output
			case 1:
				output = a.chan2.output
			case 2:
				output = a.chan3.output
			case 3:
				output = a.chan4.output
			}

			// multiply the output by the volume
			output *= uint8(a.volumes[i])

			// add the output to the left and right channels
			if a.leftEnable[i] {
				left += int(output)
			}
			if a.rightEnable[i] {
				right += int(output)
			}
		}

		// add the output to the buffer
		a.audioBuffer.sampleChan <- [2]uint16{uint16(left), uint16(right)}
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
	a.player.Pause()
}

// Play resumes the APU.
func (a *APU) Play() {
	a.playing = true
	a.enabled = true
	a.audioBuffer.start()
	a.player.Play()
}

func (a *APU) clearRegisters() {
	a.vinLeft = false
	a.vinRight = false
	a.volumeLeft = 0
	a.volumeRight = 0

	a.enabled = false

	a.leftEnable = [4]bool{false, false, false, false}
	a.rightEnable = [4]bool{false, false, false, false}
}

func (a *APU) EmptyBuffer() {
	go func() {
		for {
			<-a.audioBuffer.sampleChan
		}
	}()
}
