//go:build !test

package audio

// typedef unsigned char Uint8;
// void AudioData(void *userdata, Uint8 *stream, int len);
import "C"
import (
	"encoding/binary"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/veandco/go-sdl2/sdl"
	"os"
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
	err = initWavFile() // Create WAV file at audio start
	if err != nil {
		return err
	}
	gb = g
	sdl.PauseAudioDevice(audioDeviceID, false)

	return nil
}

func CloseAudio() error {
	err := finalizeWavFile() // Close WAV file at audio stop
	if err != nil {
		return err
	}

	sdl.CloseAudioDevice(audioDeviceID)
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

var (
	wavFile            *os.File
	wavDataSize        uint32
	wavInitialized     bool
	audioSampleRate    = 96000 // Adjust to your emulator's sample rate
	audioNumChannels   = 2     // Adjust to your emulator's number of channels (1 for mono, 2 for stereo)
	audioBitsPerSample = 32    // Adjust to your emulator's bit depth per sample (typically 8 or 16)
)

func initWavFile() error {
	var err error
	wavFile, err = os.Create("output.wav")
	if err != nil {
		return err
	}

	// Write the WAV file header placeholder
	var header [44]byte
	copy(header[:4], "RIFF")
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // PCM header size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // Audio format (1 = PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(audioNumChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(audioSampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(audioSampleRate*audioNumChannels*(audioBitsPerSample/8)))
	binary.LittleEndian.PutUint16(header[32:34], uint16(audioNumChannels*(audioBitsPerSample/8)))
	binary.LittleEndian.PutUint16(header[34:36], uint16(audioBitsPerSample))
	copy(header[36:40], "data")
	// Data chunk size placeholder (to be filled in later)
	_, err = wavFile.Write(header[:])
	if err != nil {
		return err
	}

	wavInitialized = true
	return nil
}

func finalizeWavFile() error {
	if wavFile == nil {
		return nil
	}

	// Update the file size and data chunk size in the header
	_, err := wavFile.Seek(4, 0)
	if err != nil {
		return err
	}
	if err := binary.Write(wavFile, binary.LittleEndian, uint32(36+wavDataSize)); err != nil {
		return err
	}

	_, err = wavFile.Seek(40, 0)
	if err != nil {
		return err
	}
	if err := binary.Write(wavFile, binary.LittleEndian, wavDataSize); err != nil {
		return err
	}

	err = wavFile.Close()
	if err != nil {
		return err
	}

	wavFile = nil
	return nil
}
