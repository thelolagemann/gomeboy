package tests

import (
	"context"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"github.com/thelolagemann/go-gameboy/internal/types"
	"github.com/thelolagemann/go-gameboy/pkg/log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	blarggROMPath = "roms/blargg"
)

var (
	dmgSoundTests = func() []ROMTest {
		return []ROMTest{
			newImageTest("dmg_sound/01-registers", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/02-len ctr", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/03-trigger", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/04-sweep", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/05-sweep details", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/06-overflow on trigger", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/07-len sweep period sync", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/08-len ctr during power", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/09-wave read while on", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/10-wave trigger while on", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/11-regs after power", withEmulatedSeconds(20)),
			newImageTest("dmg_sound/12-wave write while on", withEmulatedSeconds(20)),
		}
	}
	cgbSoundTests = func() []ROMTest {
		return []ROMTest{
			newImageTest("cgb_sound/01-registers", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/02-len ctr", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/03-trigger", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/04-sweep", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/05-sweep details", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/06-overflow on trigger", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/07-len sweep period sync", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/08-len ctr during power", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/09-wave read while on", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/10-wave trigger while on", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/11-regs after power", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("cgb_sound/12-wave", asModel(types.CGBABC), withEmulatedSeconds(20)),
		}
	}

	// blarggImageTests holds all the tests that are image based,
	// as they don't output any data to the 0xFF01 register
	blarggImageTests = func() []ROMTest {
		return []ROMTest{
			newImageTest("halt_bug", withEmulatedSeconds(20)),
			newImageTest("halt_bug", asModel(types.CGBABC), withEmulatedSeconds(20)),
			newImageTest("instr_timing", withEmulatedSeconds(20)),
			newImageTest("interrupt_time", withEmulatedSeconds(2)),
			newImageTest("interrupt_time", asModel(types.CGBABC), withEmulatedSeconds(2)),
		}
	}
)

// discoverROMTests discovers all of the image tests within a directory
// by looking for all files with the .gb extension and matching with
// an expected image in the same directory.
func discoverROMTests(dir string) []ROMTest {
	var tests []ROMTest

	// read the directory for all .gb/.gbc files
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".gb" && filepath.Ext(path) != ".gbc" {
			return nil
		}

		// we've found a rom, so create a test for it
		romTest := imageTest{
			romPath: path,
			model:   types.DMGABC, // default to DMG
			name:    filepath.Base(path),
		}

		// now we need to find the image the image should be in the same
		// directory as the rom and have the same name as the rom, but
		// with a .png extension, and some variation of the model name
		// appended to it e.g. 01-registers.gb -> 01-registers_dmg.png
		// or 01-registers_cgb.png
		if err := filepath.Walk(filepath.Dir(path), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".png" {
				return nil
			}

			// we've found a png, is it the one we're looking for?
			// the png should have the same name as the rom, but with a .png extension
			// and some variation of the model name appended to it

			// we've found a png with the same name as the rom, now we need to find the model
			// to run the test with

			if strings.Contains(filepath.Base(path), "cgb") {
				romTest.model = types.CGBABC
			}
			if strings.Contains(filepath.Base(path), "dmg") {
				romTest.model = types.DMGABC
			}

			romTest.expectedImage = path

			return nil
		}); err != nil {
			return err
		}

		tests = append(tests, &romTest)

		return nil
	}); err != nil {
		panic(err)
	}

	return tests
}

func Test_Blargg(t *testing.T) {
	testROMs(t, blarggImageTests()...)
	testROMs(t, dmgSoundTests()...)
	testROMs(t, cgbSoundTests()...)
}

type blarrgTest struct {
	romPath string
	name    string
	passed  bool
	model   types.Model
}

func newBlargTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(blarggROMPath, dir, "individual")
	// check if individual exists, otherwise check if rom-singles exists
	if _, err := os.Stat(romDir); os.IsNotExist(err) {
		romDir = filepath.Join(blarggROMPath, dir, "rom_singles")
	}
	tc := suite.NewTestCollection(dir)

	// read the directory
	files, err := os.ReadDir(romDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".gb" {
			continue
		}

		tc.Add(&blarrgTest{
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
		})
	}

	return tc
}

func (m *blarrgTest) Name() string {
	return m.name
}

func (m *blarrgTest) Run(t *testing.T) {
	if pass := testBlarggROM(t, m.romPath); pass {
		m.passed = true
	}
}

func (m *blarrgTest) Passed() bool {
	return m.passed
}

func testBlarrg(table *TestTable) {
	// create top level test suite
	tS := table.NewTestSuite("blarrg")

	// cgb_sound
	tS.NewTestCollection("cgb_sound").AddTests(cgbSoundTests()...)

	// cpu_instrs
	newBlargTestCollectionFromDir(tS, "cpu_instrs")
	// dmg_sound
	tS.NewTestCollection("dmg_sound").AddTests(dmgSoundTests()...)
	// halt_bug
	tS.NewTestCollection("halt_bug").AddTests(blarggImageTests()[0], blarggImageTests()[1])
	// instr_timing
	tS.NewTestCollection("instr_timing").Add(blarggImageTests()[2])
	// interrupt_time (DMG)
	tS.NewTestCollection("interrupt_time").AddTests(blarggImageTests()[3], blarggImageTests()[4])
	// mem_timing
	newBlargTestCollectionFromDir(tS, "mem_timing")
}

// testBlarggROM tests a blarrg ROM. A passing test will write
// Passed to the 0xFF01 register. A custom handler is used to intercept
// writes to the 0xFF01 register and check if the test passed.
func testBlarggROM(t *testing.T, romFile string) bool {
	passed := true
	t.Run(filepath.Base(romFile), func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romFile)
		if err != nil {
			t.Fatal(err)
		}
		output := ""
		// create the gameboy
		g := gameboy.NewGameBoy(b, gameboy.SerialDebugger(&output), gameboy.NoAudio(), gameboy.WithLogger(log.NewNullLogger()))

		// run for 10 seconds max (realtime)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		go func() {
			<-ctx.Done()
			g.CPU.DebugBreakpoint = true
		}()
		// run the gameboy
		for {
			g.Frame()
			if g.CPU.DebugBreakpoint {
				break
			}
		}

		// check if the test passed
		if strings.Contains(output, "Failed") || !strings.Contains(output, "Passed") {
			passed = false
			t.Errorf("expecting output to contain 'Passed', got '%s'", output)
		}
	})

	return passed
}

// TODO
// add way to test specific models (dmg, cgb, agb) for each pass condition (e.g. a cgb test should fail on dmg, but pass on cgb)
