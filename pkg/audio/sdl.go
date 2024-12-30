//go:build !test

package audio

// typedef unsigned char Uint8;
// void AudioData(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/veandco/go-sdl2/sdl"
	"reflect"
	"time"
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
		// output silence if gameboy is paused
		if gb.Paused() || !gb.Initialised() {
			for i := 0; i < n; i++ {
				data[i] = 0
			}
			return
		}

		frame = gb.Frame()
		s, sN := gb.APU.Samples()
		if sN > 0 {

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
	}

	copy((*[maxArraySize]byte)(frameBufferPtr)[:frameSize:frameSize], (*[maxArraySize]byte)(unsafe.Pointer(&frame[0]))[:frameSize:frameSize])
	frameBuffer <- tempFb
}

var gb *gameboy.GameBoy
var (
	frameBuffer chan []byte

	tempFb         = make([]byte, 144*160*3)
	frameBufferPtr = unsafe.Pointer(&tempFb[0])
)

func OpenAudio(g *gameboy.GameBoy, fb chan []byte) error {
	if err := sdl.AudioInit("pulsewire"); err != nil {
		if err := sdl.InitSubSystem(sdl.INIT_AUDIO); err != nil {
			return err
		}
	}
	frameBuffer = fb
	gb = g

	var err error
	if audioDeviceID, err = sdl.OpenAudioDevice("", false, &sdl.AudioSpec{
		Freq:     sampleRate,
		Format:   sdl.AUDIO_F32,
		Channels: 2,
		Samples:  bufferSize,
		Callback: sdl.AudioCallback(C.AudioData),
	}, nil, 0); err != nil {
		// if we can't initialize any audio, we need to fall back to a dummy driver
		// otherwise there won't be any output
		go func() {
			dummyBuffer := make([]C.Uint8, bufferSize*4) // Assuming 4 bytes per sample (32-bit float)
			ticker := time.NewTicker(time.Second / time.Duration(sampleRate/bufferSize))
			defer ticker.Stop()

			for range ticker.C {
				if gb != nil {
					AudioData(nil, (*C.Uint8)(unsafe.Pointer(&dummyBuffer[0])), C.int(len(dummyBuffer)))
				}
			}
		}()
		return err
	}

	sdl.PauseAudioDevice(audioDeviceID, false)

	return nil
}

var (
	audioDeviceID sdl.AudioDeviceID
)

const (
	bufferSize   = 1634
	sampleRate   = 96000
	frameSize    = 144 * 160 * 3
	maxArraySize = 144 * 160 * 4
)
