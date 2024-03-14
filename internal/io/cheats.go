package io

import (
	"fmt"
	"strconv"
	"strings"
)

type GameGenieCode struct {
	NewData byte
	Address uint16
	OldData byte
}

func parseGameGenieCode(code string) (GameGenieCode, error) {
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

func parseGameSharkCode(code string) (GameSharkCode, error) {
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
