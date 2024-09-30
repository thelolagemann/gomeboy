package tests

import (
	"fmt"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"image/png"
	"math/rand"
	"os"
	"sort"
	"testing"
)

type inputTest struct {
	expectedImagePath string
	inputs            []testInput

	*basicTest
}

func (iT *inputTest) Run(t *testing.T) {
	iT.passed = true
	t.Run(iT.name, func(t *testing.T) {
		// create a new gameboy
		gb := gameboy.NewGameBoy(gameboy.AsModel(iT.model))
		if err := gb.LoadROM(iT.romPath); err != nil {
			t.Errorf("error loading ROM: %s", err)
		}

		// sort the inputs by cycle (so we can press them in order)
		sort.Slice(iT.inputs, func(i, j int) bool {
			return iT.inputs[i].atEmulatedCycle < iT.inputs[j].atEmulatedCycle
		})

		// schedule input events on gameboy to occur at emulated cycles (with some degree of randomization TODO make configurable)
		for _, input := range iT.inputs {
			adjustedCycle := input.atEmulatedCycle
			adjustedCycle += (1024 + uint64(rand.Intn(4192))) * 4

			gb.Scheduler.ScheduleEvent(scheduler.JoypadA+scheduler.EventType(input.button), adjustedCycle)
			gb.Scheduler.ScheduleEvent(scheduler.JoypadARelease+scheduler.EventType(input.button), adjustedCycle+72240)
		}
		// wait an additional 5 seconds (60 * 5) frames to wait for test completion
		for frame := 0; frame < 60*10; frame++ {
			// get the next frame
			gb.Frame()
		}

		diff, diffImg, err := compareImage(iT.expectedImagePath, gb)
		if err != nil {
			iT.passed = false
			t.Fatal(err)
		}

		if diff > 0 {
			iT.passed = false
			t.Errorf("images are different: %d", diff) // TODO percentage

			// write output image to disk
			outFile, err := os.Create(fmt.Sprintf("results/%s_output.png", iT.name))
			if err != nil {
				t.Fatal(err)
			}
			defer outFile.Close()

			if err := png.Encode(outFile, diffImg); err != nil {
			}
		}
	})

}

type testInput struct {
	// the button to press
	button io.Button
	// the frame to press the button
	atEmulatedCycle uint64
}
