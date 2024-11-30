package io

import (
	"bufio"
	"fmt"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"io"
	"strconv"
	"strings"
)

func (b *Bus) LoadCheat(c Cheat, new bool) {
	for i, code := range c.Codes {
		switch len(code) {
		case 8:
			gCode, err := ParseGameSharkCode(code)
			if err != nil {
				c.Codes = utils.RemoveIndex(c.Codes, i)
				continue // shouldn't happen
			}
			b.GameSharkCodes = append(b.GameSharkCodes, gCode)
		case 9:
			gCode, err := ParseGameGenieCode(code)
			if err != nil {
				c.Codes = utils.RemoveIndex(c.Codes, i)
				continue // shouldn't happen
			}
			b.GameGenieCodes = append(b.GameGenieCodes, gCode)
		}
	}

	if new {
		b.LoadedCheats = append(b.LoadedCheats, c)
	}
}

func (b *Bus) UnloadCheat(id int, remove bool) error {
	if id > len(b.LoadedCheats) {
		return fmt.Errorf("cheat index %d out of bounds", id)
	}
	loadedCheat := b.LoadedCheats[id]

	if loadedCheat.Enabled {
		for _, code := range loadedCheat.Codes {
			switch len(code) {
			case 8:
				c, err := ParseGameSharkCode(code)
				if err != nil {
					continue // ignore invalid codes (although they shouldn't have been loaded in the first place >:()
				}

				// now we need to find the game shark code on the bus
				for i, gCode := range b.GameSharkCodes {
					//	fmt.Println(i, "a")
					if gCode == c {
						// to remove gameshark code, we simply remove it from the slice (as it is applied at vbl)
						b.GameSharkCodes = utils.RemoveIndex(b.GameSharkCodes, i)
						break
					}
				}
			case 9:
				c, err := ParseGameGenieCode(code)
				if err != nil {
					continue
				}

				for i, gCode := range b.GameGenieCodes {
					if gCode == c {
						// to remove a gamegenie code we also need to restore the original data
						if b.data[gCode.Address] == gCode.NewData {
							b.data[gCode.Address] = gCode.OldData
						}
						b.GameGenieCodes = utils.RemoveIndex(b.GameGenieCodes, i)
					}
				}
			}
		}
	}

	if remove {
		b.LoadedCheats = utils.RemoveIndex(b.LoadedCheats, id)
	} else {
		b.LoadedCheats[id].Enabled = false
	}

	return nil
}

type GameGenieCode struct {
	NewData byte
	Address uint16
	OldData byte
}

func ParseGameGenieCode(code string) (GameGenieCode, error) {
	if len(code) != 11 {
		return GameGenieCode{}, fmt.Errorf("invalid code format: %v", code)
	}

	code = strings.Replace(code, "-", "", -1)
	hexCodes := []string{
		code[0:2],
		code[2:6],
		code[6:7] + code[8:9],
	}

	hexCodes[1] = hexCodes[1][3:4] + hexCodes[1][0:3]
	newData, err := strconv.ParseUint(hexCodes[0], 16, 8)
	if err != nil {
		return GameGenieCode{}, err
	}
	address, err := strconv.ParseUint(hexCodes[1], 16, 16)
	if err != nil {
		return GameGenieCode{}, err
	}
	oldData, err := strconv.ParseUint(hexCodes[2], 16, 8)
	if err != nil {
		return GameGenieCode{}, err
	}

	return GameGenieCode{
		NewData: byte(newData),
		Address: uint16(address) ^ 0xF000,
		OldData: byte(oldData>>2|oldData<<6) ^ 0xBA,
	}, nil
}

type GameSharkCode struct {
	ExternalRAMBank uint8
	Address         uint16
	NewData         uint8
}

func ParseGameSharkCode(code string) (GameSharkCode, error) {
	if len(code) != 8 {
		return GameSharkCode{}, fmt.Errorf("invalid code length: %v", code)
	}

	hexCodes := []string{
		code[0:2],
		code[2:4],
		code[4:8],
	}

	hexCodes[2] = hexCodes[2][2:4] + hexCodes[2][0:2]
	ramBank, err := strconv.ParseUint(hexCodes[0], 16, 8)
	if err != nil {
		return GameSharkCode{}, err
	}
	newData, err := strconv.ParseUint(hexCodes[1], 16, 8)
	if err != nil {
		return GameSharkCode{}, err
	}
	address, err := strconv.ParseUint(hexCodes[2], 16, 16)
	if err != nil {
		return GameSharkCode{}, err
	}

	return GameSharkCode{
		ExternalRAMBank: uint8(ramBank),
		Address:         uint16(address),
		NewData:         uint8(newData),
	}, nil
}

type Cheat struct {
	Name    string
	Codes   []string
	Enabled bool
}

func ParseCheats(r io.Reader) ([]Cheat, error) {
	var cheats []Cheat
	var currentCheat *Cheat
	scanner := bufio.NewScanner(r)

	enabled := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue // skip empty lines
		}

		if strings.HasPrefix(line, "!disabled") {
			enabled = false
			continue
		}
		if strings.HasPrefix(line, "#") {
			if currentCheat != nil {
				cheats = append(cheats, *currentCheat)
			}

			currentCheat = &Cheat{
				Name:    strings.TrimSpace(line[1:]),
				Codes:   []string{},
				Enabled: enabled,
			}
			enabled = true
		} else if currentCheat != nil {
			currentCheat.Codes = append(currentCheat.Codes, line)
		}
	}

	if currentCheat != nil {
		cheats = append(cheats, *currentCheat)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cheats, nil
}
