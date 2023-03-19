package types

import (
	"os"
)

// Resettable is an interface that allows an object to be reset.
type Resettable interface {
	Reset() // Reset the state of the object
}

// State represents the Game Boy state. This is used to
// save and load states between runs.
type State struct {
	raw           []byte // raw state data (for serialization)
	readPosition  int    // current read position
	writePosition int    // current write position
}

// Stater is an interface that allows an object to be saved
// and loaded from a state.
type Stater interface {
	Load(*State) // Load the state of the object
	Save(*State) // Save the state of the object
}

// NewState creates a new state.
func NewState() *State {
	return &State{
		raw: make([]byte, 0),
	}
}

// ResetPosition resets the read and write positions,
// allowing the state to be read from the beginning.
func (s *State) ResetPosition() {
	s.readPosition = 0
	s.writePosition = 0
}

// StateFromBytes creates a new state from the given bytes.
func StateFromBytes(raw []byte) *State {
	return &State{
		raw: raw,
	}
}

func (s *State) Write8(value uint8) {
	s.raw = append(s.raw, value)
	s.writePosition++
}

func (s *State) Write16(value uint16) {
	s.raw = append(s.raw, byte(value), byte(value>>8))
	s.writePosition += 2
}

func (s *State) Write32(value uint32) {
	s.raw = append(s.raw, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
	s.writePosition += 4
}

func (s *State) WriteBool(value bool) {
	if value {
		s.raw = append(s.raw, 1)
	} else {
		s.raw = append(s.raw, 0)
	}
	s.writePosition++
}

func (s *State) WriteData(data []byte) {
	s.raw = append(s.raw, data...)
	s.writePosition += len(data)
}

func (s *State) Read8() uint8 {
	value := s.raw[s.readPosition]
	s.readPosition++
	return value
}

func (s *State) Read16() uint16 {
	value := uint16(s.raw[s.readPosition]) | uint16(s.raw[s.readPosition+1])<<8
	s.readPosition += 2
	return value
}

func (s *State) Read32() uint32 {
	value := uint32(s.raw[s.readPosition]) | uint32(s.raw[s.readPosition+1])<<8 | uint32(s.raw[s.readPosition+2])<<16 | uint32(s.raw[s.readPosition+3])<<24
	s.readPosition += 4
	return value
}

func (s *State) ReadBool() bool {
	value := s.raw[s.readPosition] != 0
	s.readPosition++
	return value
}

func (s *State) ReadData(p []byte) {
	copy(p, s.raw[s.readPosition:])
	s.readPosition += len(p)
}

func (s *State) SaveToFile(filename string) error {
	return os.WriteFile(filename, s.raw, 0644)
}

func (s *State) Bytes() []byte {
	return s.raw
}
