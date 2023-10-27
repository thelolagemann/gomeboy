//go:build !test

package audio

// typedef unsigned char Uint8;
// void AudioData(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"github.com/veandco/go-sdl2/sdl"
	"math"
	"reflect"
	"unsafe"
)

var (
	buffer = newBuffer(1024 * 1024)
	paused = false
)

// circularBuffer is a circular buffer of bytes.
type circularBuffer struct {
	buffer      []byte
	size        uint64
	readCursor  uint64
	writeCursor uint64
}

func newBuffer(size uint64) *circularBuffer {
	b := &circularBuffer{
		buffer: make([]byte, size),
		size:   size,
	}

	return b
}

func (b *circularBuffer) write(data []byte) {
	remaining := b.size - b.writeCursor
	copy(b.buffer[b.writeCursor:], data)
	if uint64(len(data)) > remaining {
		copy(b.buffer, data[remaining:])
	}

	b.writeCursor = (b.writeCursor + uint64(len(data))) % b.size
}

func (b *circularBuffer) read(data []byte) {
	if paused {
		return
	}
	n := len(data)

	if b.writeCursor-b.readCursor < bufferSize {
		// if there is less than the bufferSize available to read, then
		// we need to copy the currently buffered data, and repeat the last
		// sample until the  buffer is full again - very cursed approach to handling
		// audio desync, but it works enough to not make my ears bleed anymore :D

		// how many samples are buffered? (U16)
		samplesBuffered := (b.writeCursor - b.readCursor) / 4

		// how many samples does data want?
		samplesWanted := uint64(len(data) / 4)

		// copy the buffered samples to data
		copy(data, b.buffer[b.readCursor:b.readCursor+(samplesBuffered*4)])

		// get the last buffered sample
		bufferedSample := b.buffer[b.readCursor+(samplesBuffered*4) : b.readCursor+(samplesBuffered*4)+4]

		// for each remaining wanted sample, copy the last buffered sample
		for i := samplesBuffered; i < samplesWanted; i++ {
			copy(data[i*4:], bufferedSample)
		}

		// update the read cursor to reflect the new data
		b.readCursor = (b.readCursor + samplesBuffered*4) % b.size
		return
	}

	remaining := b.size - b.readCursor
	copy(data, b.buffer[b.readCursor:])
	if uint64(len(data)) > remaining {
		copy(data[remaining:], b.buffer)
	}

	b.readCursor = (b.readCursor + uint64(n)) % b.size
}

//export AudioData
func AudioData(userdata unsafe.Pointer, stream *C.Uint8, length C.int) {
	n := int(length)
	hdr := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(stream)), Len: n, Cap: n}
	data := *(*[]C.Uint8)(unsafe.Pointer(&hdr))

	samples := make([]byte, n)
	buffer.read(samples)

	for i, sample := range samples {
		data[i] = C.Uint8(sample)
	}
}

func OpenAudio() error {
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		return err
	}

	var err error
	if audioDeviceID, err = sdl.OpenAudioDevice("", false, &sdl.AudioSpec{
		Freq:     sampleRate,
		Format:   sdl.AUDIO_F32SYS,
		Channels: 2,
		Samples:  bufferSize,
		Callback: sdl.AudioCallback(C.AudioData),
	}, nil, 0); err != nil {
		return err
	}

	sdl.PauseAudioDevice(audioDeviceID, false)

	return nil
}

var (
	audioDeviceID sdl.AudioDeviceID
)

const (
	bufferSize = 128
	sampleRate = 96000
)

func PlaySDL(data []float32) {
	if !paused {
		var b []byte
		for i := range data {
			n := math.Float32bits(data[i])
			b = append(b, byte(n), byte(n>>8))
			b = append(b, byte(n>>16), byte(n>>24))
		}

		buffer.write(b)
	}
}

func Pause() {
	paused = true
	sdl.PauseAudioDevice(audioDeviceID, true)
}

func Play() {
	paused = false
	sdl.PauseAudioDevice(audioDeviceID, false)
}
