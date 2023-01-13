package display

import (
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"github.com/thelolagemann/go-gameboy/internal/ppu/palette"
	"image/color"
	"math"
)

// PixelScale is the multiplier for the pixel size.
var PixelScale float64 = 8

type Display struct {
	window  *pixelgl.Window
	picture *pixel.PictureData
}

func NewDisplay(title string) *Display {
	cfg := pixelgl.WindowConfig{
		Title: "GomeBoy | " + title,
		Bounds: pixel.R(
			0, 0,
			float64(ppu.ScreenWidth*PixelScale), float64(ppu.ScreenHeight*PixelScale)),
		VSync:     true,
		Resizable: true,
	}

	// create a new window
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// create a new picture
	picture := pixel.MakePictureData(pixel.R(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))

	// create a new display
	d := &Display{
		window:  win,
		picture: picture,
	}

	// update camera and return newly created display
	d.updateCamera()

	return d
}

// Render renders the given frame to the display.
func (d *Display) Render(frame [160][144][3]uint8) {
	for y := 0; y < ppu.ScreenHeight; y++ {
		for x := 0; x < ppu.ScreenWidth; x++ {
			r := frame[x][y][0]
			g := frame[x][y][1]
			b := frame[x][y][2]
			d.picture.Pix[(ppu.ScreenHeight-1-y)*ppu.ScreenWidth+x] = color.RGBA{R: r, G: g, B: b, A: 255}
		}
	}

	// get background colour
	rgb := palette.GetColour(3)
	bg := color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}
	d.window.Clear(bg)

	sprite := pixel.NewSprite(pixel.Picture(d.picture), pixel.R(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))
	sprite.Draw(d.window, pixel.IM)

	d.updateCamera()
	d.window.Update()
}

// updateCamera updates the camera position.
func (d *Display) updateCamera() {
	xScale := d.window.Bounds().W() / ppu.ScreenWidth
	yScale := d.window.Bounds().H() / ppu.ScreenHeight
	scale := math.Min(yScale, xScale)

	shift := d.window.Bounds().Size().Scaled(0.5).Sub(pixel.ZV)
	cam := pixel.IM.Scaled(pixel.ZV, scale).Moved(shift)
	d.window.SetMatrix(cam)
}

// SetTitle sets the title of the window.
func (d *Display) SetTitle(title string) {
	d.window.SetTitle(fmt.Sprintf("GomeBoy | %s", title))
}

var keyMap = map[pixelgl.Button]Input{
	pixelgl.KeyA:         joypad.ButtonA,
	pixelgl.KeyS:         joypad.ButtonB,
	pixelgl.KeyEnter:     joypad.ButtonStart,
	pixelgl.KeyBackspace: joypad.ButtonSelect,
	pixelgl.KeyRight:     joypad.ButtonRight,
	pixelgl.KeyLeft:      joypad.ButtonLeft,
	pixelgl.KeyUp:        joypad.ButtonUp,
	pixelgl.KeyDown:      joypad.ButtonDown,

	pixelgl.KeyC:           CyclePalette,
	pixelgl.KeyP:           Pause,
	pixelgl.KeyLeftControl: Speedup,
}

type Inputs struct {
	Pressed, Released []Input
}

type Input = uint8

const (
	// CyclePalette changes the palette
	CyclePalette Input = iota + 8
	Pause
	Speedup
)

// PollKeys polls the keys and returns the pressed and released keys.
func (d *Display) PollKeys() Inputs {
	var pressed, released []joypad.Button

	for key, button := range keyMap {
		if d.window.JustPressed(key) {
			pressed = append(pressed, button)
		}
		if d.window.JustReleased(key) {
			released = append(released, button)
		}
	}

	return Inputs{Pressed: pressed, Released: released}
}
