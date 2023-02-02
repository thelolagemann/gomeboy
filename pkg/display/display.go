package display

import (
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/thelolagemann/go-gameboy/internal/joypad"
	"github.com/thelolagemann/go-gameboy/internal/ppu"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"os"
)

// PixelScale is the multiplier for the pixel size.
var PixelScale float64 = 8

const (
	ScreenWidth  = 574
	ScreenHeight = 949
)

type Display struct {
	window  *pixelgl.Window
	picture *pixel.PictureData
}

func (d *Display) IsClosed() bool {
	return d.window.Closed()
}

func NewDisplay(title string) *Display {
	cfg := pixelgl.WindowConfig{
		Title: "GomeBoy | " + title,
		Bounds: pixel.R(
			0, 0,
			float64(ppu.ScreenWidth), float64(ppu.ScreenHeight)),
		VSync:                  true,
		TransparentFramebuffer: true,
		Resizable:              true,
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
	//win.Canvas().SetUniform("uDotSize", &uDotSize)
	//win.Canvas().SetUniform("uDotSpacing", &uDotSpacing)
	//win.Canvas().SetUniform("uWindowWidth", &uWindowWidth)
	//win.Canvas().SetUniform("uWindowHeight", &uWindowHeight)

	return d
}

func (d *Display) ApplyShader(shader string, uniforms map[string]interface{}) {
	for name, value := range uniforms {
		d.window.Canvas().SetUniform(name, value)
	}
	d.window.Canvas().SetFragmentShader(shader)
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

	// clear the window
	d.window.Clear(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	// draw frame from 142, 120 to 443, 396 ( so 301x276 = 301/160 = 1.88, 276/144 = 1.92) TODO fix dimensions so they are a multiple of 160x144
	sprite := pixel.NewSprite(d.picture, pixel.R(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))
	sprite.Draw(d.window, pixel.IM)

	// get background colour
	// rgb := palette.GetColour(3)
	// bg := color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}
	// d.window.Clear(bg)

	// sprite := pixel.NewSprite(pixel.Picture(d.picture), pixel.R(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))
	// sprite.Draw(d.window, pixel.IM)
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
	pixelgl.KeyLeftControl: DumpTilemap,
}

type Inputs struct {
	Pressed, Released []Input
}

type Input = uint8

const (
	// CyclePalette changes the palette
	CyclePalette Input = iota + 8
	Pause
	DumpTilemap
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

	i := Inputs{pressed, released}

	return i
}

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return pixel.PictureDataFromImage(img), nil
}

// createCartImage creates an image of the Cartridge by composing
// the blank cartridge frame at /frames/cart.png with image located
// at /labels/<MD5>.png. If the label image does not exist, a blank
// cartridge image is returned.
func createCartImage(md5 string) pixel.Picture {
	frame, err := os.Open("frames/cart.png")
	if err != nil {
		log.Fatal(err)
	}
	defer frame.Close()

	frameImg, _, err := image.Decode(frame)
	if err != nil {
		log.Fatal(err)
	}

	label, err := os.Open(fmt.Sprintf("labels/%s.png", md5))
	if err != nil {
		return pixel.PictureDataFromImage(frameImg)
	}
	defer label.Close()

	labelImg, _, err := image.Decode(label)
	if err != nil {
		return pixel.PictureDataFromImage(frameImg)
	}

	// compose the images
	// the label should be horizontally centered and offset
	// vertically by -578 pixels TODO ensure this is correct
	//offset := image.Pt(-frameImg.Bounds().Dx()/2+labelImg.Bounds().Dx()/2, -578)
	b := frameImg.Bounds()
	m := image.NewRGBA(b)

	// draw the frame
	draw.Draw(m, b, frameImg, image.Point{}, draw.Src)

	// scale the label with Catmull-Rom resampling
	// and preserve its aspect ratio
	// the label should be drawn from
	f := image.NewRGBA(image.Rect(0, 0, 1270, 1120))
	draw.CatmullRom.Scale(f, f.Bounds(), labelImg, labelImg.Bounds(), draw.Src, nil)

	// draw the label
	draw.Draw(m, f.Bounds().Add(image.Point{X: 220, Y: 570}), f, image.Point{}, draw.Over)

	return pixel.PictureDataFromImage(m)
}
