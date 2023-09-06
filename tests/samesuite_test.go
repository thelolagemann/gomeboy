package tests

import (
	"os"
	"path/filepath"
)

const (
	samesuiteROMPath = "roms/same-suite"
)

func newSamesuiteTestCollectionFromDir(suite *TestSuite, dir string) *TestCollection {
	// more or less the same as mooneye as it uses the same pass/fail mechanism
	romDir := filepath.Join(samesuiteROMPath, dir)
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

		tc.Add(&mooneyeTest{
			romPath: filepath.Join(romDir, file.Name()),
			name:    file.Name(),
		})
	}

	return tc
}

func testSamesuite(roms *TestTable) {
	// create top level test suite
	tS := roms.NewTestSuite("samesuite")

	// apu
	newSamesuiteTestCollectionFromDir(tS, "apu")

	// dma
	newSamesuiteTestCollectionFromDir(tS, "dma")

	// interrupt
	newSamesuiteTestCollectionFromDir(tS, "interrupt")

	// ppu
	newSamesuiteTestCollectionFromDir(tS, "ppu")

	// sgb
	newSamesuiteTestCollectionFromDir(tS, "sgb")
}
