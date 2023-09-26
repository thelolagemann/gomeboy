package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/ppu"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"image"
	"image/color"
)

type Render struct {
	Video *ppu.PPU
}

func (r Render) Title() string {
	return "Render"
}

func (r Render) Run(w fyne.Window, events <-chan event.Event) error {
	// create the base image
	img := image.NewRGBA(image.Rect(0, 0, 160, 144))

	c := container.NewVBox()
	// create a canvas to display the image
	cImg := canvas.NewImageFromImage(img)
	cImg.ScaleMode = canvas.ImageScalePixels
	cImg.FillMode = canvas.ImageFillOriginal

	cImg.Resize(fyne.NewSize(160*4, 140*4))

	// create a legend for the colors
	legend := container.NewGridWithRows(4)

	// create a 32x32 image for each color
	red := image.NewRGBA(image.Rect(0, 0, 32, 32))
	orange := image.NewRGBA(image.Rect(0, 0, 32, 32))
	green := image.NewRGBA(image.Rect(0, 0, 32, 32))
	blue := image.NewRGBA(image.Rect(0, 0, 32, 32))

	// fill the images with the colors
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			red.Set(x, y, color.RGBA{255, 0, 0, 255})
			orange.Set(x, y, color.RGBA{255, 165, 0, 255})
			green.Set(x, y, color.RGBA{0, 255, 0, 255})
			blue.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}

	// create the canvas images
	redImg := canvas.NewImageFromImage(red)
	orangeImg := canvas.NewImageFromImage(orange)
	greenImg := canvas.NewImageFromImage(green)
	blueImg := canvas.NewImageFromImage(blue)

	// resize the images
	redImg.Resize(fyne.NewSize(32, 32))
	redImg.FillMode = canvas.ImageFillOriginal
	orangeImg.Resize(fyne.NewSize(32, 32))
	orangeImg.FillMode = canvas.ImageFillOriginal
	greenImg.Resize(fyne.NewSize(32, 32))
	greenImg.FillMode = canvas.ImageFillOriginal
	blueImg.Resize(fyne.NewSize(32, 32))
	blueImg.FillMode = canvas.ImageFillOriginal

	// add the images to the legend
	legend.Add(container.NewHBox(redImg, widget.NewLabel("Sprite Change Made Line Dirty")))
	legend.Add(container.NewHBox(orangeImg, widget.NewLabel("Sprite Change Made Pixel Dirty")))
	legend.Add(container.NewHBox(greenImg, widget.NewLabel("Sprite Tile Could Make Line Dirty")))
	legend.Add(container.NewHBox(blueImg, widget.NewLabel("Sprite On Line Could Make Line Dirty")))

	// add the legend to the container
	c.Add(legend)
	c.Add(cImg)

	// create a box for the event viewer
	eventBox := container.NewVBox()

	// add the event box to the container
	c.Add(eventBox)

	// set the content of the window to the canvas
	w.SetContent(c)
	go func() {
		for {
			// get the next event
			e := <-events

			// check if the event is a frame event
			if e.Type == event.FrameTime {
				// update the image
				r.Video.DumpRender(img)

				// redraw the canvas
				c.Refresh()
			} else if e.Type == event.Quit {
				return
			}
		}
	}()
	return nil
}
