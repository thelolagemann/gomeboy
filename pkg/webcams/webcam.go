package webcams

import (
	"errors"
)

type Webcam interface {
	StartStreaming() error
	StopStreaming() error
	ReadFrame() ([]byte, error)
	Close() error
	Device() string
	Name() string
}

var (
	errUnsupportedOS = errors.New("webcams: unsupported OS")
)
