package gameboy

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func romTestWalker(t *testing.T) fs.WalkDirFunc {
	return func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".gb" {
			t.Run(path, func(t *testing.T) {
				testRom(t, path)
			})
		}

		return nil
	}
}

func testRom(t *testing.T, romPath string) {
	// run rom
	b, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGameBoy(b, NoBios())
	g.Start(nil)

	// wait until cpu hits debug breakpoint
	ti := time.NewTicker(time.Second * 1)
f:
	for {
		select {
		case <-ti.C:
			if g.CPU.DebugPause {
				break f
			}
		}
	}

	// compare output
	// a passing test writes the fibonacci sequence 3/5/8/13/21/34 to the registers B/C/D/E/H/L
	// a failing test writes the byte 0x42 to the registers B/C/D/E/H/L
	if g.CPU.B == 0x42 {
		t.Fatal("test failed")
	}
}

func Test(t *testing.T) {
	// test mooneye roms
	// get roms in folder and subfolders
	if err := filepath.WalkDir("./roms/mooneye", romTestWalker(t)); err != nil {
		t.Fatal(err)
	}

	// for each rom
	// run rom
	// compare output
}
