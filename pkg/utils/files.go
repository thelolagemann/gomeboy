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
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	// does the file have an extension?
	if filepath.Ext(filename) == "" {
		return data, nil
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
	data, err = io.ReadAll(decoder)

	return data, nil
}

func Unzip(zipFile, destFolder string) error {
	// Open the ZIP file for reading
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// Create the destination folder if it doesn't exist
	if err := os.MkdirAll(destFolder, os.ModePerm); err != nil {
		return err
	}

	// Iterate through the files in the ZIP archive
	for _, file := range r.File {
		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// Construct the path to the file in the destination folder
		destPath := filepath.Join(destFolder, file.Name)

		if file.FileInfo().IsDir() {
			// If it's a directory, create it in the destination folder
			os.MkdirAll(destPath, os.ModePerm)
		} else {
			// If it's a file, create the necessary directories and extract the file
			if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
				return err
			}

			destFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer destFile.Close()

			_, err = io.Copy(destFile, rc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
