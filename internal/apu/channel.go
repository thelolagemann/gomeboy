package apu

import (
	"math"
)

// A Channel represents a single audio channel.
type Channel struct {
	frequency float64
	generator WaveGenerator
	time      float64
	amplitude float64

	// duration in samples
	duration int
	length   int

	envVolume    int
	envTime      int
	envSteps     int
	envStepsInit int
	envSamples   int
	envIncrease  bool

	sweepTime     float64
	sweepStepLen  byte
	sweepSteps    byte
	sweepStep     byte
	sweepIncrease bool

	onL bool
	onR bool
}

// NewChannel returns a new channel.
func NewChannel() *Channel {
	return &Channel{}
}

// Sample returns a single sample for streaming the
// audio output. Each sample will include the left and
// right channel. Each channel is an signed 16-bit integer.
func (c *Channel) Sample() (outputL, outputR uint16) {
	var output uint16
	step := c.frequency * twoPi / sampleRate
	c.time += step
	if c.shouldPlay() {
		output = uint16(float64(c.generator(c.time)) * c.amplitude)
	}

	if c.duration > 0 {
		c.duration--
	}
	c.updateEnvelope()
	c.updateSweep()
	if c.onL {
		outputL = output
	}
	if c.onR {
		outputR = output
	}
	return
}

// shouldPlay returns true if the channel should play.
func (c *Channel) shouldPlay() bool {
	return (c.duration == -1 || c.duration > 0) &&
		c.generator != nil && c.envStepsInit > 0
}

// updateEnvelope updates the envelope of the channel.
func (c *Channel) updateEnvelope() {
	if c.envSamples > 0 {
		c.envTime += 1
		if c.envSteps > 0 && c.envTime >= c.envSamples {
			c.envTime -= c.envSamples
			c.envSteps--

			if c.envSteps == 0 {
				c.amplitude = 0
			} else if c.envIncrease {
				c.amplitude = 1 - float64(c.envSteps)/float64(c.envStepsInit)
			} else {
				c.amplitude = float64(c.envSteps) / float64(c.envStepsInit)
			}
		}
	}
}

var sweepTimes = map[byte]float64{
	1: 7.8,
	2: 15.6,
	3: 23.4,
	4: 31.3,
	5: 39.1,
	6: 46.9,
	7: 54.7,
}

// updateSweep updates the sweep of the channel.
func (c *Channel) updateSweep() {
	if c.sweepStep < c.sweepSteps {
		t := sweepTimes[c.sweepStepLen] / 1000
		c.sweepTime += perSample

		if c.sweepTime > t {
			c.sweepTime -= t
			c.sweepStep++

			if c.sweepIncrease {
				c.frequency += c.frequency / math.Pow(2, float64(c.sweepStep))
			} else {
				c.frequency -= c.frequency / math.Pow(2, float64(c.sweepStep))
			}
		}
	}
}

// Reset resets the channel.
func (c *Channel) Reset(duration int) {
	c.amplitude = 1
	c.envTime = 0
	c.sweepTime = 0
	c.sweepStep = 0
	c.duration = duration
}
