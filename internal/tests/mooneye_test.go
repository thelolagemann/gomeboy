package mooneye

import (
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMooneye_Bits(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/bits/mem_oam.gb")
	testMooneyeROM(t, "roms/acceptance/bits/reg_f.gb")
	testMooneyeROM(t, "roms/acceptance/bits/unused_hwio-GS.gb")
}

func TestMooneye_Instr(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/instr/daa.gb")
}

func TestMooneye_Interrupts(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/interrupts/ie_push.gb")
}

func TestMooneye_OAM_DMA(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/oam_dma/basic.gb")
	testMooneyeROM(t, "roms/acceptance/oam_dma/reg_read.gb")
	testMooneyeROM(t, "roms/acceptance/oam_dma/sources-GS.gb")
}

func TestMooneye_PPU(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/ppu/hblank_ly_scx_timing-GS.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/intr_1_2_timing-GS.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/intr_2_0_timing.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/intr_2_mode0_timing.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/intr_2_mode0_timing_sprites.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/intr_2_mode3_timing.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/intr_2_oam_ok_timing.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/lcdon_timing-GS.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/stat_irq_blocking.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/stat_lyc_onoff.gb")
	testMooneyeROM(t, "roms/acceptance/ppu/vblank_stat_intr-GS.gb")
}

func TestMooneye_Serial(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/serial/boot_sclk_align-dmgABCmgb.gb")
}

func TestMooneye_Timer(t *testing.T) {
	testMooneyeROM(t, "roms/acceptance/timer/div_write.gb")
	testMooneyeROM(t, "roms/acceptance/timer/rapid_toggle.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim00.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim00_div_trigger.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim01.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim01_div_trigger.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim10.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim10_div_trigger.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim11.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tim11_div_trigger.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tima_reload.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tima_write_reloading.gb")
	testMooneyeROM(t, "roms/acceptance/timer/tma_write_reloading.gb")
}

func TestMooneye_Individual(t *testing.T) {
	// test all of the loose files in acceptance folder
	files, err := os.ReadDir("roms/acceptance")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		testMooneyeROM(t, filepath.Join("roms/acceptance", file.Name()))
	}
}

// testMooneyeROM tests a mooneye rom. A passing test will
// execute the rom until the breakpoint is reached (LD B, B),
// and writes the fibonacci sequence 3/5/8/13/21/34 to the
// registers B, C, D, E, H, L. The test will then compare the
// registers to the expected values.
func testMooneyeROM(t *testing.T, romFile string) {

	t.Run(filepath.Base(romFile), func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romFile)
		if err != nil {
			panic(err)
		}

		// load boot rom
		boot, err := os.ReadFile("boot/dmg_boot.bin")
		if err != nil {
			panic(err)
		}

		// create the gameboy
		g := gameboy.NewGameBoy(b, gameboy.Debug(), gameboy.WithBootROM(boot))

		takenTooLong := false
		go func() {
			// run the gameboy for 10 seconds TODO figure out how long it should take
			time.Sleep(3 * time.Second)
			takenTooLong = true
		}()
		// run until breakpoint
		for {
			g.Frame()
			if g.CPU.DebugBreakpoint || takenTooLong {
				break
			}
		}

		// check the registers
		if g.CPU.B != 3 {
			t.Errorf("B register is %d, expected 3", g.CPU.B)
		}
		if g.CPU.C != 5 {
			t.Errorf("C register is %d, expected 5", g.CPU.C)
		}
		if g.CPU.D != 8 {
			t.Errorf("D register is %d, expected 8", g.CPU.D)
		}
		if g.CPU.E != 13 {
			t.Errorf("E register is %d, expected 13", g.CPU.E)
		}
		if g.CPU.H != 21 {
			t.Errorf("H register is %d, expected 21", g.CPU.H)
		}
		if g.CPU.L != 34 {
			t.Errorf("L register is %d, expected 34", g.CPU.L)
		}
	})
}
