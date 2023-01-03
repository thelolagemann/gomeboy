package io

type Serial struct {
	output uint8
}

func NewSerial() *Serial {
	return &Serial{}
}

func (s *Serial) Read() uint8 {
	return s.output
}

func (s *Serial) Write(value uint8) {
	s.output = value
}
