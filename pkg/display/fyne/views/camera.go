package views

import (
	"bytes"
	_ "embed"
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"github.com/thelolagemann/gomeboy/pkg/webcams"
	"golang.org/x/image/bmp"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Camera struct {
	widget.BaseWidget

	*io.Camera
	*ppu.PPU

	webcams        []webcams.Webcam
	webcamSelect   *widget.Select
	selectedWebcam string

	exposureLabel *widget.Label
	shotImage     *canvas.Image
	sensedImage   *canvas.Image
	tiledImage    *canvas.Image
}

func NewCamera(cam *io.Camera, p *ppu.PPU) *Camera {
	c := &Camera{Camera: cam, PPU: p}
	c.ExtendBaseWidget(c)

	return c
}

func (c *Camera) getNames() []string {
	var names []string

	for _, web := range c.webcams {
		names = append(names, web.Name()+" ("+web.Device()+")")
	}
	return names
}

func (c *Camera) loadCams() {
	for _, cam := range c.webcams {
		if err := cam.Close(); err != nil {
			log.Errorf("error closing webcam: %v", err)
		}
	}
	cams, err := webcams.FindAllWebcams()
	if err != nil {
		log.Errorf("error discovering webcams: %v", err)
		return
	}

	c.webcams = cams
	c.selectedWebcam = ""
	c.webcamSelect.Selected = ""
	c.webcamSelect.Options = c.getNames()
	c.webcamSelect.Refresh()
	log.Debugf("reloaded cameras %v %s %d", c.webcamSelect.Options, c.webcamSelect.Selected, len(c.webcams))
}

var (
	providers = []string{"None", "Image", "Noise", "Webcam"}
	//go:embed "none.png"
	blankImageBytes []byte
	blankImage, _   = png.Decode(bytes.NewReader(blankImageBytes))
	blankImageFn    = func() image.Image { return blankImage }
)

func (c *Camera) WebcamShooter(cam webcams.Webcam) (io.CameraShooter, func()) {
	if err := cam.StartStreaming(); err != nil {
		log.Errorf("could not start webcam: %v", err)
		return blankImageFn, nil
	}

	var lastImg image.Image
	var mu sync.Mutex
	return func() image.Image {
			mu.Lock()
			defer mu.Unlock()
		readCam:
			f, err := cam.ReadFrame()
			if err != nil || len(f) == 0 {
				if lastImg == nil {
					return blankImage
				}
				var errno syscall.Errno
				if errors.As(err, &errno) && errors.Is(errno, syscall.ENODEV) {
					log.Debugf("camera disconnected")
					c.loadCams()
					c.CameraShooter = blankImageFn
					return blankImage
				} else if errors.As(err, &errno) && errors.Is(errno, syscall.EAGAIN) {
					goto readCam
				}

				return lastImg
			}
			img, err := jpeg.Decode(bytes.NewReader(f))
			if err != nil {
				log.Errorf("could not decode jpeg image: %v", err)
				return blankImage
			}
			if img == nil {
				return blankImage
			}
			lastImg = img
			return img
		}, func() {
			mu.Lock()
			defer mu.Unlock()
			if err := cam.StopStreaming(); err != nil {
				log.Errorf("could not stop webcam: %v", err) // probably cam disconnect
				c.loadCams()
			}
		}
}

func PictureShooter(pic string) io.CameraShooter {
	f, err := os.Open(pic)
	if err != nil {
		log.Errorf("error opening picture %s %v", pic, err)
		return blankImageFn
	}
	defer f.Close()

	var img image.Image
	switch filepath.Ext(pic) {
	case ".bmp":
		img, err = bmp.Decode(f)
	case ".png":
		img, err = png.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	}

	if err != nil {
		log.Errorf("error decoding picture %s %v", pic, err)
		return blankImageFn
	}

	return func() image.Image { return img }
}

func (c *Camera) CreateRenderer() fyne.WidgetRenderer {
	contentBox := container.NewVBox()

	selectedImage := ""
	tearDown := func() {}
	c.selectedWebcam = ""
	c.webcamSelect = widget.NewSelect(c.getNames(), func(s string) {
		if c.selectedWebcam == s {
			return
		}
		if tearDown != nil {
			tearDown()
		}
		c.selectedWebcam = s
		for _, cam := range c.webcams {
			if strings.Contains(s, cam.Device()) {
				c.CameraShooter, tearDown = c.WebcamShooter(cam)
			}
		}
	})
	c.loadCams()

	refreshButton := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if tearDown != nil {
			tearDown()
			tearDown = nil
		}
		c.loadCams()
	})
	refreshButton.Resize(fyne.NewSize(40, 40))

	providerContent := map[string]fyne.CanvasObject{
		"Image": widget.NewButtonWithIcon("Select Image", theme.FileImageIcon(), func() {
			d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err != nil {
					return
				}
				if reader == nil {
					return // user cancelled
				}

				selectedImage = reader.URI().Path()
				c.CameraShooter = PictureShooter(selectedImage)
			}, findWindow("Camera"))
			d.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg"}))
			d.Show()
		}),
		"Webcam": container.NewBorder(nil, nil, nil, refreshButton, c.webcamSelect),
	}

	main := container.NewVBox()
	selectedProvider := "None"
	providerDropdown := widget.NewSelect(providers, func(s string) {
		if s == selectedProvider {
			return
		}
		selectedProvider = s
		if tearDown != nil {
			tearDown()
			tearDown = nil
		}

		switch s {
		case "None":
			c.CameraShooter = func() image.Image { return blankImage }
		case "Noise":
			c.CameraShooter = io.NoiseShooter(io.CameraSensorW, io.CameraSensorH)
		case "Image":
			if selectedImage == "" {
				c.CameraShooter = func() image.Image { return blankImage }
			} else {
				c.CameraShooter = PictureShooter(selectedImage)
			}
		case "Webcam":
			for _, cam := range c.webcams {
				if strings.Contains(c.webcamSelect.Selected, cam.Device()) {
					c.CameraShooter, tearDown = c.WebcamShooter(cam)
				}
			}
		}

		if content, ok := providerContent[s]; ok {
			contentBox.Objects = []fyne.CanvasObject{content}
		} else {
			contentBox.Objects = []fyne.CanvasObject{}
		}
	})

	main.Add(container.NewHBox(bold("Camera Source"), providerDropdown))
	main.Add(contentBox)

	// webcams options
	cameraOptions := container.NewVBox(bold("Camera Settings"))

	// enable/disable filtering options
	filterOptions := container.NewHBox(bold("Filtering"))
	filter1D := widget.NewCheck("1D", func(b bool) { c.Filter1D = b })
	filter1D.SetChecked(true)
	filter1DAndHoriz := widget.NewCheck("1D & Horiz Enhancement", func(b bool) { c.Filter1DAndHoriz = b })
	filter1DAndHoriz.SetChecked(true)
	filter2D := widget.NewCheck("2D", func(b bool) { c.Filter2D = b })
	filter2D.SetChecked(true)
	filterOptions.Add(filter1D)
	filterOptions.Add(filter1DAndHoriz)
	filterOptions.Add(filter2D)
	cameraOptions.Add(filterOptions)

	// actual webcams settings
	c.exposureLabel = widget.NewLabel(strconv.Itoa(exposureTime(uint16(c.Camera.Registers[io.CameraC1]) | uint16(c.Camera.Registers[io.CameraC0])<<8)))
	cameraOptions.Add(container.NewHBox(bold("Exposure"), c.exposureLabel))
	main.Add(cameraOptions)

	// camera views
	c.shotImage = canvas.NewImageFromImage(c.ShotImage)
	c.shotImage.SetMinSize(fyne.NewSize(128*4, 112*4))

	c.shotImage.FillMode = canvas.ImageFillContain

	c.shotImage.ScaleMode = canvas.ImageScalePixels

	c.sensedImage = canvas.NewImageFromImage(image.NewGray(image.Rect(0, 0, 128, 120)))
	c.sensedImage.SetMinSize(fyne.NewSize(128*4, 120*4))
	c.sensedImage.FillMode = canvas.ImageFillContain
	c.sensedImage.ScaleMode = canvas.ImageScalePixels
	c.tiledImage = canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 128, 112)))
	c.tiledImage.SetMinSize(fyne.NewSize(128*4, 112*4))
	c.tiledImage.FillMode = canvas.ImageFillContain
	c.tiledImage.ScaleMode = canvas.ImageScalePixels

	images := container.NewHBox(c.shotImage, c.sensedImage, c.tiledImage)

	main.Add(images)

	return widget.NewSimpleRenderer(main)
}

func (c *Camera) Refresh() {
	c.exposureLabel.SetText("1/" + strconv.Itoa(exposureTime(uint16(c.Camera.Registers[io.CameraC1])|uint16(c.Camera.Registers[io.CameraC0])<<8)) + "s")

	c.shotImage.Image = c.ShotImage
	c.shotImage.Refresh()

	for y := 0; y < io.CameraSensorH; y++ {
		for x := 0; x < io.CameraSensorW; x++ {
			c.sensedImage.Image.(*image.Gray).Pix[y*io.CameraSensorW+x] = uint8(c.SensedImage[x][y])
		}
	}

	var x, y = 0, 0
	drawTiles := func(from, to int) {
		for tile := from; tile < to; tile++ {
			if x == 128 { // end of line
				y += 8
				x = 0
			}
			c.PPU.TileData[0][tile].Draw(c.tiledImage.Image.(*image.RGBA), x, y, c.PPU.ColourPalette[0])
			x += 8
		}
	}
	drawTiles(256, 384)
	drawTiles(128, 144)
	drawTiles(0, 80)
	c.tiledImage.Refresh()
	c.sensedImage.Refresh()
}

func exposureTime(e uint16) int { return 1048576 / (int(e) * 16) }
