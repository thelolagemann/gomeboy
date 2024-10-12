package webcams

import (
	"fmt"
	"github.com/blackjack/webcam"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type linuxWebcam struct {
	*webcam.Webcam

	device, name string
}

func (l linuxWebcam) Device() string { return l.device }
func (l linuxWebcam) Name() string   { return l.name }

func FindAllWebcams() ([]Webcam, error) {
	var cams []Webcam
	err := filepath.WalkDir("/dev", func(path string, info fs.DirEntry, err error) error {
		if !strings.Contains(path, "video") || info.IsDir() || err != nil {
			return nil // skip non-video, dirs & errors
		}
		if info.Type()&os.ModeCharDevice != 0 {
			c, err := webcam.Open(path)
			if err != nil {
				return nil // device unsupported, skip to next
			}
			if err := c.SetBufferCount(1); err != nil {
				return fmt.Errorf("error configuring webcam device: %s %v", path, err)
			}
			name, err := c.GetName()
			if err != nil {
				return fmt.Errorf("error getting name: %v", err)
			}

			cams = append(cams, linuxWebcam{device: path, name: name, Webcam: c})
		}
		return nil
	})
	if len(cams) == 0 && err != nil {
		return nil, err
	}

	return cams, nil
}
