package tests

import (
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/scheduler"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"
)

type inputTest struct {
	expectedImagePath string
	inputs            []testInput

	*basicTest
}

func (iT *inputTest) Run(t *testing.T) {
	iT.passed = true
	t.Run(iT.name, func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(iT.romPath)
		if err != nil {
			t.Fatal(err)
		}

		// create a new gameboy
		gb := gameboy.NewGameBoy(b, gameboy.AsModel(iT.model), gameboy.Speed(0), gameboy.NoAudio(), gameboy.WithLogger(log.NewNullLogger()))

		// setup frame, event and input channels
		frames := make(chan []byte, 144)
		events := make(chan event.Event, 144)
		pressed := make(chan io.Button, 10)
		released := make(chan io.Button, 10)

		// sort the inputs by cycle (so we can press them in order)
		sort.Slice(iT.inputs, func(i, j int) bool {
			return iT.inputs[i].atEmulatedCycle < iT.inputs[j].atEmulatedCycle
		})

		var lastCycle uint64
		// schedule input events on gameboy to occur at emulated cycles (with some degree of randomization TODO make configurable)
		for _, input := range iT.inputs {
			adjustedCycle := input.atEmulatedCycle
			adjustedCycle += (1024 + uint64(rand.Intn(4192))) * 4

			gb.Scheduler.ScheduleEvent(scheduler.JoypadA+scheduler.EventType(input.button), adjustedCycle)
			gb.Scheduler.ScheduleEvent(scheduler.JoypadARelease+scheduler.EventType(input.button), adjustedCycle+72240)
			lastCycle = adjustedCycle + 72240
		}

		done := make(chan struct{}, 2)
		go func() {
			// wait for the cycle
			for gb.Scheduler.Cycle() < lastCycle {
				time.Sleep(time.Millisecond * 10)
			}
			done <- struct{}{}
			done <- struct{}{}
		}()

		// empty event channel
		go func() {
			for {
				select {
				case <-events:
				}
			}
		}()

		go func() {
			for {
				select {
				case <-done:
					return
				case <-frames:
				}
			}
		}()

		// start the gameboy
		go func() {
			gb.Start(frames, events, pressed, released)
		}()

		// wait for the test to finish
		<-done
		// wait an additional 5 seconds (60 * 5) frames to wait for test completion
		for frame := 0; frame < 60*5; frame++ {
			// get the next frame
			<-frames
		}

		// close the channels
		close(done)

		diff, _, err := compareImage(iT.expectedImagePath, gb)
		if err != nil {
			iT.passed = false
			t.Fatal(err)
		}

		if diff > 0 {
			iT.passed = false
			t.Errorf("images are different: %d", diff) // TODO percentage
		}
	})

}

type testInput struct {
	// the button to press
	button io.Button
	// the frame to press the button
	atEmulatedCycle uint64
}
