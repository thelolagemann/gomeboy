package io

import "fmt"

type Serial struct {
	output uint8
}

func NewSerial() *Serial {
	return &Serial{}
}

func (s *Serial) Read(address uint16) uint8 {
	switch address {
	case 0xFF01:
		return s.output
	case 0xFF02:
		return 0x7E
	default:
		panic(fmt.Sprintf("serial\tillegal read from address %04X", address))
	}
}

func (s *Serial) Write(address uint16, value uint8) {
	if address == 0xFF01 {
		s.output = value
	} else {
		panic(fmt.Sprintf("serial\tillegal write to address %04X", address))
	}
}
