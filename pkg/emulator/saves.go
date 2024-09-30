package emulator

import (
	"os"
)

var ramSizes = []uint32{
	0x0000,   // 0KB RAM
	0x0800,   // 2KB RAM
	0x2000,   // 8KB RAM
	0x8000,   // 32KB RAM
	0x20000,  // 128KB RAM
	0x100000, // 64KB RAM
}

// Save represents a save file.
type Save struct {
	b    []byte   // the save file data
	f    *os.File // temporary file that is written to when the emu is running
	Path string   // the path to the save file
}

// NewSave creates a new save file for the given cartridge title,
// and RAM size.
func NewSave(title string, ramSize uint) (*Save, error) {
	// does the sav file already exist?
	if _, err := os.Stat(title + ".sav"); err == nil {
		return LoadSave(title)
	}

	// create the file
	f, err := os.Create(title + ".sav")
	if err != nil {
		return nil, err
	}

	s := Save{
		b:    make([]byte, ramSize),
		f:    f,
		Path: title + ".sav",
	}

	// return the save file
	return &s, nil
}

// LoadSave loads the save file for the given cartridge title.
func LoadSave(cartTitle string) (*Save, error) {
	f, err := os.OpenFile(cartTitle+".sav", os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	s := Save{
		b:    make([]byte, info.Size()),
		f:    f,
		Path: cartTitle + ".sav",
	}

	// read the save file data
	if _, err := s.f.ReadAt(s.b, 0); err != nil {
		return nil, err
	}

	return &s, nil
}

// Bytes returns the save file data.
func (s *Save) Bytes() []byte {
	return s.b
}

// SetBytes sets the save file data.
func (s *Save) SetBytes(b []byte) error {
	s.b = b
	return nil
}

// Close closes the save file by renaming the temporary file to the original file.
func (s *Save) Close() error {
	if s.f == nil {
		return nil
	}

	// write bytes to file
	if _, err := s.f.WriteAt(s.b, 0); err != nil {
		return err
	}
	if err := s.f.Close(); err != nil {
		return err
	}
	return os.Rename(s.f.Name(), s.Path)
}
