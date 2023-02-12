package tests

import (
	"bytes"
	"fmt"
	"github.com/thelolagemann/go-gameboy/internal/gameboy"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func Test_All(t *testing.T) {
	testTable := &TestTable{
		testSuites: make([]*TestSuite, 0),
	}
	//testAcid2(t, testTable)
	//testBlarrg(t, testTable)
	testMooneye(t, testTable)
	//testSamesuite(t, testTable)

	// execute tests
	for _, top := range testTable.testSuites {
		t.Run(top.name, func(t *testing.T) {
			for _, collection := range top.collections {
				t.Run(collection.name, func(t *testing.T) {
					for _, test := range collection.tests {
						test.Run(t)
					}
				})
			}
		})
	}

	// wait for all tests to finish

	// write markdown table to README.md
	f, err := os.Create("README.md")
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.WriteString(testTable.CreateReadme())
}

func ImgCompare(img1, img2 image.Image) (int64, image.Image, error) {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	if bounds1 != bounds2 {
		return math.MaxInt64, nil, fmt.Errorf("image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
	}

	accumError := int64(0)
	resultImg := image.NewRGBA(image.Rect(
		bounds1.Min.X,
		bounds1.Min.Y,
		bounds1.Max.X,
		bounds1.Max.Y,
	))
	draw.Draw(resultImg, resultImg.Bounds(), img1, image.Point{0, 0}, draw.Src)

	for x := bounds1.Min.X; x < bounds1.Max.X; x++ {
		for y := bounds1.Min.Y; y < bounds1.Max.Y; y++ {
			r1, g1, b1, a1 := img1.At(x, y).RGBA()
			r2, g2, b2, a2 := img2.At(x, y).RGBA()

			diff := int64(sqDiffUInt32(r1, r2))
			diff += int64(sqDiffUInt32(g1, g2))
			diff += int64(sqDiffUInt32(b1, b2))
			diff += int64(sqDiffUInt32(a1, a2))

			if diff > 0 {
				accumError += diff
				resultImg.Set(
					bounds1.Min.X+x,
					bounds1.Min.Y+y,
					color.RGBA{R: 255, A: 255})
			}
		}
	}

	return int64(math.Sqrt(float64(accumError))), resultImg, nil
}

func sqDiffUInt32(x, y uint32) uint64 {
	d := uint64(x) - uint64(y)
	return d * d
}

// TestTable is a collection of many TestSuite(s).
type TestTable struct {
	// Top level tests
	testSuites []*TestSuite
}

func (t *TestTable) CreateReadme() string {
	// create the table of contents with links
	tableOfContents := "# Table of Contents\n"
	for _, suite := range t.testSuites {
		tableOfContents += "* [" + suite.name + "](#" + suite.name + ")\n"
		for _, collection := range suite.collections {
			tableOfContents += "  * [" + collection.name + "](#" + collection.name + ")\n"
		}
	}

	// create the test results
	table := ""
	for _, suite := range t.testSuites {
		table += "# " + suite.name + "\n"
		for _, collection := range suite.collections {
			table += "## " + collection.name + "\n"
			table += CreateMarkdownTableFromTests(collection.tests)
		}
	}

	return tableOfContents + table
}

// TestSuite is a collection of tests (often by a single author, or for a single
// feature) that can be run together.
type TestSuite struct {
	name        string
	collections []*TestCollection
}

func (t *TestSuite) NewTestCollection(name string) *TestCollection {
	collection := &TestCollection{name: name, tests: make([]ROMTest, 0)}
	t.collections = append(t.collections, collection)
	return collection
}

func (t *TestSuite) NewTestCollectionFromDir(dir string) *TestCollection {
	collection := &TestCollection{name: dir, tests: make([]ROMTest, 0)}
	t.collections = append(t.collections, collection)
	return collection
}

func (t *TestTable) NewTestSuite(name string) *TestSuite {
	suite := &TestSuite{name: name, collections: make([]*TestCollection, 0)}
	t.testSuites = append(t.testSuites, suite)
	return suite
}

type TestCollection struct {
	tests []ROMTest
	name  string
}

func (t *TestCollection) Add(test ROMTest) {
	t.tests = append(t.tests, test)
}

type ROMTest interface {
	Run(t *testing.T)
	Passed() bool
	Name() string
}

func CreateMarkdownTableFromTests(tests []ROMTest) string {
	table := "| Test | Passing |\n| ---- | ------- |\n"
	for _, test := range tests {
		// pass is green check, fail is red x
		pass := "✅"
		if !test.Passed() {
			pass = "❌"
		}
		table += "| " + test.Name() + " | " + pass + " |\n"
		fmt.Println(test.Name(), test.Passed())
	}
	return table
}

func testROMWithExpectedImage(t *testing.T, romPath string, expectedImagePath string, asModel gameboy.Model) bool {
	passed := true
	t.Run(filepath.Base(romPath), func(t *testing.T) {
		// load the rom
		b, err := os.ReadFile(romPath)
		if err != nil {
			t.Fatal(err)
		}

		// create the emulator
		g := gameboy.NewGameBoy(b, gameboy.AsModel(asModel))

		// run for 60 seconds
		for i := 0; i < 60*60; i++ {
			g.Frame()
		}

		// get the current frame
		img := g.Frame()

		// create image.Image from the byte array
		img1 := image.NewRGBA(image.Rect(0, 0, 160, 144))
		for y := 0; y < 144; y++ {
			for x := 0; x < 160; x++ {
				img1.Set(x, y, color.RGBA{
					R: img[y][x][0],
					G: img[y][x][1],
					B: img[y][x][2],
					A: 255,
				})
			}
		}

		// compare the image to the expected image
		expectedImg, err := os.ReadFile(expectedImagePath)
		if err != nil {
			t.Fatal(err)
		}
		img2, _, err := image.Decode(bytes.NewReader(expectedImg))
		if err != nil {
			t.Fatal(err)
		}

		// compare the images
		diff, resultImg, err := ImgCompare(img1, img2)
		if err != nil {
			t.Fatal(err)
		}

		// if the images are different, save the result image
		if diff > 0 {
			t.Errorf("images are different by %d pixels", diff)
			passed = false
			// save the result image
			f, err := os.Create("results/" + filepath.Base(romPath) + "_result.png")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			png.Encode(f, resultImg)
		}
	})
	return passed
}

// TODO:
// - add a way to run tests in parallel
// - perform rom tests with expected image output
// - parse description from test roms (maybe)
