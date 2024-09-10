//go:build !test

package audio

// typedef unsigned char Uint8;
// void AudioData(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/veandco/go-sdl2/sdl"
	"reflect"
	"unsafe"
)

var sampleBuffer []uint8
var frame [144][160][3]uint8

//export AudioData
func AudioData(userdata unsafe.Pointer, stream *C.Uint8, length C.int) {
	n := int(length)
	hdr := reflect.SliceHeader{Data: uintptr(unsafe.Pointer(stream)), Len: n, Cap: n}
	data := *(*[]C.Uint8)(unsafe.Pointer(&hdr))

	// if we already have a frame's worth of buffered samples then no need to step
	if len(sampleBuffer) > n {
		for i := 0; i < n; i++ {
			data[i] = C.Uint8(sampleBuffer[i])
		}
		sampleBuffer = sampleBuffer[n:]
	} else {
		//	start := time.Now()

		frame = gb.Frame()
		s, sN := gb.APU.Samples()

		samples := append(sampleBuffer, unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), len(s)*4)...)
		for i := 0; i < n; i++ {
			if uint32(i) < sN*4 {
				data[i] = C.Uint8(samples[i])
			}
		}

		if len(samples) > n {
			sampleBuffer = samples[n:]
		} else {
			sampleBuffer = sampleBuffer[0:]
		}
	}

	copy((*[maxArraySize]byte)(frameBufferPtr)[:frameSize:frameSize], (*[maxArraySize]byte)(unsafe.Pointer(&frame[0]))[:frameSize:frameSize])
	frameBuffer <- tempFb
}

var gb *gameboy.GameBoy
var (
	frameBuffer chan []byte
	events      chan event.Event

	tempFb         = make([]byte, 144*160*3)
	frameBufferPtr = unsafe.Pointer(&tempFb[0])
)

func OpenAudio(g *gameboy.GameBoy, fb chan []byte, e chan event.Event) error {
	if err := sdl.AudioInit("pulsewire"); err != nil {
		if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
			return err
		}
	}
	frameBuffer = fb
	events = e

	var err error
	if audioDeviceID, err = sdl.OpenAudioDevice("", false, &sdl.AudioSpec{
		Freq:     sampleRate,
		Format:   sdl.AUDIO_F32,
		Channels: 2,
		Samples:  bufferSize,
		Callback: sdl.AudioCallback(C.AudioData),
	}, nil, 0); err != nil {
		return err
	}

	gb = g
	sdl.PauseAudioDevice(audioDeviceID, false)

	return nil
}

var (
	audioDeviceID sdl.AudioDeviceID
)

const (
	bufferSize = 1634
	sampleRate = 96000
)

const (
	frameSize    = 144 * 160 * 3
	maxArraySize = 144 * 160 * 4
)
