package tests

import (
	"github.com/thelolagemann/go-gameboy/internal/types"
	"testing"
)

var (
	// timer00_to_01
	timer00_to_01 = mooneyeTest{
		name:    "timer_00_to_01",
		romPath: "roms/misc/timer_00_to_01_trigger.gb",
		model:   types.DMGABC,
	}
	// wx_split tests the behaviour of the WX register when it is set to 7
	wx_split = newImageTest("wx_split", withEmulatedSeconds(5))
)

func Test_Misc(t *testing.T) {
	// wx_split.Run(t)

	timer00_to_01.Run(t)
}
