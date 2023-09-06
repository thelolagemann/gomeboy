//go:build !test

package audio

// typedef unsigned char Uint8;
// void AudioData(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"github.com/veandco/go-sdl2/sdl"
	"reflect"
	"unsafe"
)

var (
	buffer   = newBuffer(1024 * 1024)
	paused   = false
	disabled = false
)

// circularBuffer is a circular buffer of bytes.
type circularBuffer struct {
	buffer                  []byte
	size                    uint64
	readCursor              uint64
	writeCursor             uint64
	bytesRead, bytesWritten uint64
}

func newBuffer(size uint64) *circularBuffer {
	b := &circularBuffer{
		buffer: make([]byte, size),
		size:   size,
	}

	return b
}

func (b *circularBuffer) write(data []byte) {
	n := len(data)
	b.bytesWritten += uint64(n)

	remaining := b.size - b.writeCursor
	copy(b.buffer[b.writeCursor:], data)
	if uint64(len(data)) > remaining {
		copy(b.buffer, data[remaining:])
	}

	b.writeCursor = (b.writeCursor + uint64(len(data))) % b.size
}

func (b *circularBuffer) read(data []byte) {
	if paused || b.bytesWritten == 0 {
		return
	}
	n := len(data)
	b.bytesRead += uint64(n)

	remaining := b.size - b.readCursor
	copy(data, b.buffer[b.readCursor:])
	if uint64(len(data)) > remaining {
		copy(data[remaining:], b.buffer)
	}

	// fmt.Println(b.writeCursor, b.readCursor, len(data), b.bytesRead, b.bytesWritten)

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
		Format:   sdl.AUDIO_U16SYS,
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
	bufferSize = 256
	sampleRate = 44100
)

func PlaySDL(data []byte) {
	if !disabled {
		buffer.write(data)
	}
}

func Pause() {
	paused = true
}

func Play() {
	paused = false
}
