package emu

import (
	"fmt"
	"github.com/thelolagemann/go-gameboy/pkg/utils"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	saveFolder = "saves" // TODO make this configurable
)

var ramSizes = []uint32{
	0x0000,   // 0KB RAM
	0x0800,   // 2KB RAM
	0x2000,   // 8KB RAM
	0x8000,   // 32KB RAM
	0x20000,  // 128KB RAM
	0x100000, // 64KB RAM
}

// save file naming convention:
// <MD5 Cart Checksum>.<timestamp>.sav

// TODO
// -- save with timestamp
// -- load newest save file
// -- each cartridge gets its own folder in the save folder , e.g. "saves/roms/<MD5 Cart Checksum>" as well as a metadata file
// -- named saves (e.g. "saves/roms/<MD5 Cart Checksum>/<name>.sav") so that you can have multiple save files for the same emu
// -- -- in the metadata file, there is a list of save files with their names and timestamps, mapped by their checksums
// -- -- by default, the newest save file is loaded
// -- write to .tmp file when emu is running, then rename to .sav when emu is closing (or when the user saves manually)
// -- -- this way, if the emu crashes, the save file is not corrupted  (https://stackoverflow.com/a/2333872/13181681)

// Save represents a save file.
type Save struct {
	b    []byte   // the save file data
	f    *os.File // temporary file that is written to when the emu is running
	Path string   // the path to the save file
}

// NewSave creates a new save file for the given cartridge title,
// and RAM size.
func NewSave(title string, ramSize uint) (*Save, error) {
	// create the save folder for the cartridge if it doesn't exist
	romSaveFolder := filepath.Join(saveFolder, title)
	if err := os.MkdirAll(romSaveFolder, 0755); err != nil {
		return nil, err
	}

	// create the save file with the current timestamp and given RAM size
	timeStamp := time.Now().Unix()
	fileName := fmt.Sprintf("%d.sav", timeStamp)
	filePath := filepath.Join(romSaveFolder, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	if err := f.Truncate(int64(ramSize)); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	// create the initial save file data
	s := Save{
		b:    make([]byte, ramSize),
		Path: filePath,
	}
	if err := s.createTemporarySaveFile(); err != nil {
		return nil, err
	}

	// return the save file
	return &s, nil
}

// LoadSaves loads all save files for the given cartridge title.
// The save files are sorted by their last modified time, with the
// newest save file being the first in the slice. If no save files
// exist, an empty slice is returned.
func LoadSaves(cartTitle string) ([]*Save, error) {
	romSaveFolder := filepath.Join(saveFolder, cartTitle)

	// does the folder exist?
	if _, err := os.Stat(romSaveFolder); os.IsNotExist(err) {
		// create the save folder, and return an empty slice (no save files)
		if err := os.MkdirAll(romSaveFolder, 0755); err != nil {
			return nil, err
		}
		return make([]*Save, 0), nil
	}

	// get the save files
	files, err := os.ReadDir(romSaveFolder)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return make([]*Save, 0), nil
	}

	// sort the save files by last modified time
	sort.Slice(files, func(i, j int) bool {
		iInfo, _ := files[i].Info()
		jInfo, _ := files[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	saves := make([]*Save, 0)

	// load save files
	for _, file := range files {
		if !isFileSaveFile(file.Name()) {
			continue
		}
		savePath := filepath.Join(romSaveFolder, file.Name())
		b, err := utils.LoadFile(savePath)
		if err != nil {
			return nil, err
		}
		saves = append(saves, &Save{b: b, Path: savePath})
	}

	return saves, nil
}

// Bytes returns the save file data and opens a temporary file
// for writing if it doesn't exist.
func (s *Save) Bytes() []byte {
	if s.f == nil {
		if err := s.createTemporarySaveFile(); err != nil {
			panic(err)
		}
	}
	return s.b
}

// SetBytes sets the save file data.
func (s *Save) SetBytes(b []byte) error {
	s.b = b
	// write the data to the temporary file
	if _, err := s.f.WriteAt(b, 0); err != nil {
		return fmt.Errorf("failed to write to temporary save file: %w", err)
	}
	return nil
}

// Close closes the save file by renaming the temporary file to the original file.
func (s *Save) Close() error {
	if s.f == nil {
		return nil
	}
	if err := s.f.Close(); err != nil {
		return err
	}
	return os.Rename(s.f.Name(), s.Path)
}

// createTemporarySaveFile creates the temporary save file for the caller.
func (s *Save) createTemporarySaveFile() error {
	var err error
	s.f, err = os.CreateTemp(filepath.Dir(s.Path), fmt.Sprintf("%s.*", filepath.Base(s.Path)))
	if err != nil {
		return err
	}
	return s.f.Truncate(int64(len(s.b)))
}

// parseTimestampFromFilename parses the timestamp from the given filename.
// The filename is expected to be in the format of "<...>.<timestamp>.sav".
// Where <timestamp> is the number of seconds since the Unix epoch,
// and <...> is any string.
func parseTimestampFromFilename(filename string) int64 {
	// strip the file extension
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// get the timestamp from the filename (the last part preceded by a dot)
	parts := strings.Split(filename, ".")
	n, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// isFileSaveFile returns true if the given filename is a save file. (TODO use heuristics to determine if it's a save file)
func isFileSaveFile(filename string) bool {
	return strings.HasSuffix(filename, ".sav")
}
