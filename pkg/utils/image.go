//go:build !test

package utils

import (
	"bytes"
	"github.com/sqweek/dialog"
	"golang.design/x/clipboard"
	"image"
	"image/png"
	"os"
)

func CopyImage(img image.Image) error {
	err := clipboard.Init()
	if err != nil {
		return err
	}

	// encode image to byte slice
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return err
	}

	clipboard.Write(clipboard.FmtImage, b.Bytes())

	return nil
}

func SaveImage(img image.Image) error {
	// ask user where to save the image
	filename, err := dialog.File().Filter("PNG Image", "png").Title("Save Image").Save()
	if err != nil {
		return err
	}

	// does file have a .png extension?
	if len(filename) < 4 || filename[len(filename)-4:] != ".png" {
		filename += ".png"
	}

	// save the image
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	// TODO add more formats
	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	return nil
}
