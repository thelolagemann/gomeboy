package tests

import (
	"fmt"
	"os"
	"testing"
)

func Test_All(t *testing.T) {
	testTable := &TestTable{
		testSuites: make([]*TestSuite, 0),
	}
	testMooneye(t, testTable)
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

// TODO:
// - add a way to run tests in parallel
// - perform rom tests with expected image output
// - parse description from test roms (maybe)
