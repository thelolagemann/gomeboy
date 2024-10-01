package tests

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"image/png"
	"math/rand/v2"
	"os"
	"testing"
	"time"
)

type inputTest struct {
	expectedImagePath string
	*basicTest
}

// Retry retries the provided function fn until it succeeds or the maximum retries is reached.
// It returns a func(t *testing.T) that can be used in test cases.
func Retry(fn func() error, retries int) func(t *testing.T) {
	return func(t *testing.T) {
		for i := 0; i < retries; i++ {
			err := fn()
			if err == nil {
				return // Test passed
			}

		}
		t.Fatalf("Test failed after %d attempts", retries)
	}
}

func (iT *inputTest) Run(t *testing.T) {
	iT.passed = true
	t.Run(iT.name, Retry(func() error {
		// create a new gameboy
		gb := gameboy.NewGameBoy(gameboy.AsModel(iT.model))
		if err := gb.LoadROM(iT.romPath); err != nil {
			return err
		}

		// run a second worth of frames for setup
		for i := 0; i < 55; i++ {
			gb.Frame()
		}

		var testFinished = false
		go func() {
			for i := io.ButtonA; i <= io.ButtonDown; i++ {
				for !gb.Running() {
				}
				for i := 0; i < rand.IntN(512); i++ {
				} // burn a random amount of time
				gb.Bus.Press(i)

				time.Sleep(time.Millisecond * 20)
				for !gb.Running() {
				}
				gb.Bus.Release(i)
				time.Sleep(time.Millisecond * 10)
			}
			testFinished = true
		}()
		for !testFinished {
			// get the next frame
			gb.Frame()
			time.Sleep(time.Millisecond * 5) // give some time for the input handler
		}
		for i := 0; i < 60*10; i++ {
			gb.Frame()
		}

		diff, diffImg, err := compareImage(iT.expectedImagePath, gb)
		if err != nil {
			iT.passed = false
			t.Fatal(err)
		}

		if diff > 0 {
			iT.passed = false

			outFile, err := os.Create(fmt.Sprintf("results/%s_output.png", iT.name))
			if err != nil {
			}
			defer outFile.Close()

			if err := png.Encode(outFile, diffImg); err != nil {
			}
			// write output image to disk
			return fmt.Errorf("images are different: %d", diff) // TODO percentage
		}
		return nil
	}, 5))

}

type testInput struct {
	// the button to press
	button io.Button
	// the frame to press the button
	atEmulatedCycle uint64
}
