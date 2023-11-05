package tests

import (
	"github.com/thelolagemann/gomeboy/internal/types"
	"os"
	"path/filepath"
	"testing"
)

const (
	ageROMPath = "roms/age"
)

// assertModelsPassed is a helper function to assert which models
// should pass a test given the filename.
//
// e.g.
//
//	ei-halt-dmgC-cgbBCE.gb should pass on DMGABC and CGBABC
func assertModelsPassed(file os.DirEntry) []types.Model {
	// get name
	name := file.Name()

	// ends with cgbE -> should pass on CGBE
	if name[len(name)-len("cgbE.gb"):] == "cgbE.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBE
	}

	// ends with ncmE -> should pass on CGBE (non CGB mode)
	if name[len(name)-len("ncmE.gb"):] == "ncmE.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBE
	}

	// ends with ncmBC -> should pass on CGBBC (non CGB mode)
	if name[len(name)-len("ncmBC.gb"):] == "ncmBC.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBBC
	}

	// ends with cgbBC -> should pass on CGBBC
	if name[len(name)-len("cgbBC.gb"):] == "cgbBC.gb" {
		return []types.Model{types.CGBABC} // TODO correctly differentiate between CGBABC and CGBBC
	}

	// ends with dmgC-cgbBC -> should pass on DMGABC and CGBBC
	if name[len(name)-len("dmgC-cgbBC.gb"):] == "dmgC-cgbBC.gb" {
		return []types.Model{types.DMGABC, types.CGBABC} // TODO correctly differentiate between CGBABC and CGBBC
	}

	// ends with dmgC-cgbBCE -> should pass on DMGABC and CGBABC
	if name[len(name)-len("dmgC-cgbBCE.gb"):] == "dmgC-cgbBCE.gb" {
		return []types.Model{types.DMGABC, types.CGBABC}
	}

	// default to DMGABC
	return []types.Model{types.DMGABC}
}

func newAgeTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	romDir := filepath.Join(ageROMPath, dir)
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

		// get models that should pass
		models := assertModelsPassed(file)

		// create test for each model
		for _, model := range models {
			tc.AddTests(&mooneyeTest{basicTest: newBasicTest(filepath.Join(romDir, file.Name()), model)})
		}
	}

	return tc
}

func TestAge(t *testing.T) {
	// create top level test
	//tS := table.NewTestSuite("age")
	table := &TestTable{}
	tS := table.NewTestSuite("age")
	// halt
	newAgeTestCollectionFromDir(tS, "halt").Run(t)
	// lcd-align-ly
	newAgeTestCollectionFromDir(tS, "lcd-align-ly").Run(t)
	// ly
	newAgeTestCollectionFromDir(tS, "ly").Run(t)
	// oam
	newAgeTestCollectionFromDir(tS, "oam").Run(t)
	// stat-interrupt
	newAgeTestCollectionFromDir(tS, "stat-interrupt").Run(t)
	// stat-mode
	newAgeTestCollectionFromDir(tS, "stat-mode").Run(t)
	// stat-mode-sprites
	newAgeTestCollectionFromDir(tS, "stat-mode-sprites").Run(t)
	// stat-mode-window
	newAgeTestCollectionFromDir(tS, "stat-mode-window").Run(t)
	// vram
	newAgeTestCollectionFromDir(tS, "vram").Run(t)
}

func testAge(t *TestTable) {
	// create top level test
	tS := t.NewTestSuite("age")

	// halt
	newAgeTestCollectionFromDir(tS, "halt")
	// lcd-align-ly
	newAgeTestCollectionFromDir(tS, "lcd-align-ly")
	// ly
	newAgeTestCollectionFromDir(tS, "ly")
	// oam
	newAgeTestCollectionFromDir(tS, "oam")
	// stat-interrupt
	newAgeTestCollectionFromDir(tS, "stat-interrupt")
	// stat-mode
	newAgeTestCollectionFromDir(tS, "stat-mode")
	// stat-mode-sprites
	newAgeTestCollectionFromDir(tS, "stat-mode-sprites")
	// stat-mode-window
	newAgeTestCollectionFromDir(tS, "stat-mode-window")
	// vram
	newAgeTestCollectionFromDir(tS, "vram")
}
