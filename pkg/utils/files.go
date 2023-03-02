package utils

import (
	"archive/zip"
	"compress/gzip"
	"github.com/bodgit/sevenzip"
	"io"
	"os"
	"path/filepath"
)

// LoadFile loads the given file and performs decompression if necessary.
func LoadFile(filename string) ([]byte, error) {
	// open the file
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read the file into a byte slice
	data := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := f.Read(buf)
		if err != nil {
			break
		}
		data = append(data, buf[:n]...)
	}

	// is the file compressed?
	if filename[len(filename)-3:] == ".gb" || filename[len(filename)-4:] == ".gbc" {
		return data, nil
	}

	// is it a boot rom?
	if (len(data) == 256 || len(data) == 2304) && filename[len(filename)-4:] == ".bin" {
		return data, nil
	}

	// try to assert the compression type from the file extension
	var decoder io.Reader
	switch ext := filepath.Ext(filename); ext {
	case ".gz":
		decoder, err = gzip.NewReader(f)
	case ".xz":
	// decoder, err = xz.NewReader(f)
	case ".zip":
		// open the zip file
		zipReader, err := zip.NewReader(f, int64(len(data)))
		if err != nil {
			return nil, err
		}

		// read the first file in the zip file
		zipFile := zipReader.File[0]

		// open the file in the zip file
		decoder, err = zipFile.Open()
		if err != nil {
			return nil, err
		}
	case ".7z":
		r, err := sevenzip.NewReader(f, int64(len(data)))
		if err != nil {
			return nil, err
		}

		// read the first file in the archive
		zipFile := r.File[0]

		// open the file in the archive
		decoder, err = zipFile.Open()
	default:
		// return the data as is
		return data, nil
	}

	if err != nil {
		return nil, err
	}

	// read the decompressed data into a byte slice
	data = make([]byte, 0)
	buf = make([]byte, 1024)

	for {
		n, err := decoder.Read(buf)
		if err != nil {
			break
		}
		data = append(data, buf[:n]...)
	}

	return data, nil
}
