package cheats

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseCheatFile parses a cheat file and populates the given
// GameGenie and GameShark structs. The file format is as follows:
//
//	# Cheat Name
//	12345678
//	12345678
//	12345678
//
// Cheat files may have any number of GameGenie and GameShark codes, and
// may be mixed together. The GameGenie and GameShark structs are
// populated with the codes in the order they are found in the file.
func ParseCheatFile(filename string, genie *GameGenie, shark *GameShark) ([]Cheat, error) {
	// open the file
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	// create a scanner
	scanner := bufio.NewScanner(f)

	var cheats []Cheat
	var currentCheat *Cheat

	// read the file and populate the structs
	for scanner.Scan() {
		// read the line
		line := scanner.Text()

		// if it's a comment, read the name
		if line[0] == '#' { // once we have a name, we can start parsing codes
			cheats = append(cheats, Cheat{
				Name: line[1:],
			})
			// set the current cheat
			currentCheat = &cheats[len(cheats)-1]
			continue
		}

		// if it's a code, parse it
		if len(line) == 11 { // GameGenie
			// parse the code
			if err := genie.Load(line, currentCheat.Name); err != nil {
				return nil, err
			}

			// add the code to the current cheat
			currentCheat.codes = append(currentCheat.codes, line)

			continue
		}
		if len(line) == 8 { // GameShark
			if err := shark.Load(line, currentCheat.Name); err != nil {
				return nil, err
			}

			// add the code to the current cheat
			currentCheat.codes = append(currentCheat.codes, line)
			continue
		}
		// if it's empty, continue
		// if it's invalid, return an error
		return nil, fmt.Errorf("invalid code: %s", line)
	}

	return cheats, nil
}

func ParseCheatText(text, name string, genie *GameGenie, shark *GameShark) error {
	// create a scanner
	scanner := bufio.NewScanner(strings.NewReader(text))

	// read the file and populate the structs
	for scanner.Scan() {
		// read the line
		line := scanner.Text()

		// if it's a code, parse it
		if len(line) == 11 { // GameGenie
			// parse the code
			if err := genie.Load(line, name); err != nil {
				return err
			}

			continue
		}
		if len(line) == 8 { // GameShark
			if err := shark.Load(line, name); err != nil {
				return err
			}

			continue
		}
		// if it's empty, continue
		// if it's invalid, return an error
		return fmt.Errorf("invalid code: %s", line)
	}

	return nil
}

// SaveCheatFile saves the given Cheats to the given file.
func SaveCheatFile(filename string, cheats []Cheat) error {
	// open the file
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	// write the codes
	for _, c := range cheats {
		_, err := f.WriteString(fmt.Sprintf("# %s\n", c.Name))
		if err != nil {
			return err
		}
		for _, code := range c.codes {
			_, err := f.WriteString(fmt.Sprintf("%s\n", code))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type Cheat struct {
	Name    string
	Enabled bool

	codes []string
}
