package utils

import (
	"bytes"
	"golang.design/x/clipboard"
	"image"
	"image/png"
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
