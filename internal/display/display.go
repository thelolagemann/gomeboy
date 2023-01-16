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
	"time"
)

// PixelScale is the multiplier for the pixel size.
var PixelScale float64 = 1

const (
	ScreenWidth  = 574
	ScreenHeight = 949
)

type Display struct {
	window  *pixelgl.Window
	picture *pixel.PictureData

	background *pixel.Sprite
	button     *pixel.Sprite
	action     *pixel.Sprite
	direction  *pixel.Sprite
	cart       *pixel.Sprite
	label      *pixel.Sprite

	debounce *buttonDebouncer
}

func (d *Display) IsClosed() bool {
	return d.window.Closed()
}

type buttonDebouncer struct {
	framesActive map[joypad.Button]uint8
	isReleased   map[joypad.Button]bool
}

func (b *buttonDebouncer) update() {
	for button, frames := range b.framesActive {
		if b.isReleased[button] {
			if frames > 0 {
				b.framesActive[button] = frames - 1
			}
			if frames == 0 {
				b.remove(button)
				delete(b.isReleased, button)
			}

		}
	}
}

func (b *buttonDebouncer) isDebounced(button joypad.Button) bool {
	_, ok := b.framesActive[button]
	return ok
}

func (b *buttonDebouncer) setDebounced(button joypad.Button, frames uint8) {
	b.framesActive[button] = frames
}

func (b *buttonDebouncer) remove(button joypad.Button) {
	delete(b.framesActive, button)
}

func NewDisplay(title string, md5sum string) *Display {
	cfg := pixelgl.WindowConfig{
		Title: "GomeBoy | " + title,
		Bounds: pixel.R(
			0, 0,
			float64(ScreenWidth), float64(ScreenHeight)),
		VSync:                  true,
		TransparentFramebuffer: true,
		Resizable:              true,
	}

	// create a new window
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// load background
	pic, err := loadPicture("frames/background.png")
	if err != nil {
		panic(err)
	}

	sprite := pixel.NewSprite(pic, pic.Bounds())

	// load buttons
	pic, err = loadPicture("frames/button.png")
	if err != nil {
		panic(err)
	}

	button := pixel.NewSprite(pic, pic.Bounds())

	// load action
	pic, err = loadPicture("frames/action.png")
	if err != nil {
		panic(err)
	}

	action := pixel.NewSprite(pic, pic.Bounds())

	// load direction
	pic, err = loadPicture("frames/direction.png")
	if err != nil {
		panic(err)
	}

	direction := pixel.NewSprite(pic, pic.Bounds())

	label := createCartImage(md5sum)

	// create a new picture
	picture := pixel.MakePictureData(pixel.R(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))

	// create a new display
	d := &Display{
		window:     win,
		picture:    picture,
		background: sprite,
		button:     button,
		action:     action,
		direction:  direction,
		label:      pixel.NewSprite(label, label.Bounds()),
		debounce: &buttonDebouncer{
			framesActive: make(map[joypad.Button]uint8),
			isReleased:   make(map[joypad.Button]bool),
		},
	}

	// update camera and return newly created display
	d.updateCamera()

	return d
}

// RenderBootAnimation renders the boot animation.
func (d *Display) RenderBootAnimation() {
	for i := 0; i < 120; i++ {
		// clear window and draw background
		d.window.Clear(color.RGBA{0, 0, 0, 255})
		d.background.Draw(d.window, pixel.IM)

		// draw cart image
		d.label.Draw(d.window, pixel.IM.Scaled(pixel.ZV, 0.2).Moved(pixel.V(0, float64(-i*2))))

		// update window and camera
		d.window.Update()

		// wait for 1/60th of a second
		time.Sleep(time.Second / 60)
	}
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
	sprite.Draw(d.window, pixel.IM.Scaled(pixel.ZV, 1.92).Moved(pixel.ZV.Add(pixel.V(0, 221))))

	// draw background
	d.background.Draw(d.window, pixel.IM)

	// draw buttons
	for button := joypad.Button(0); button <= 8; button++ {
		if !d.debounce.isDebounced(button) {
			continue
		}
		switch button {
		// Button A is drawn at 462, 568 and has a size of 68x68 pixels.
		// the button will be drawn to the center of the screen, so needs
		// to be offset by 949 / 2 = (474) 474 is the center of the screen
		// and needs to be drawn at 568 so the y coordinate is 568 - 474 + (68/2) = 128
		// the x coordine is 574 / 2 = (287) 287 is the center of the screen
		// and needs to be drawn at 462 so the x coordinate is 462 - 287 + (68/2) = 137
		case joypad.ButtonA:
			d.button.Draw(d.window, pixel.IM.Moved(pixel.V(209, -128)))

		// Button B is drawn at 369, 610 and has a size of 68x68 pixels.
		// 949 /2 = 474, 610 - 474 + (68/2) = 170
		// 574 /2 = 287, 369 - 287 + (68/2) = 95
		case joypad.ButtonB:
			d.button.Draw(d.window, pixel.IM.Moved(pixel.V(116, -170)))
		// 175, 746 and has a size of 65x41 pixels.
		// 949 /2 = 474, 746 - 474 + (41/2) = 292.5
		// 574 /2 = 287, 175 - 287 + (65/2) = 79.5
		case joypad.ButtonSelect:
			d.action.Draw(d.window, pixel.IM.Moved(pixel.V(-79.5, -292.5)))
		// 271, 746 and has a size of 65x41 pixels.
		// 949 /2 = 474, 746 - 474 + (41/2) = 292.5
		// 574 /2 = 287, 271 - 287 + (65/2) = 16.5
		case joypad.ButtonStart:
			d.action.Draw(d.window, pixel.IM.Moved(pixel.V(16.5, -292.5)))
			// 92, 561 and has a size of 51x45 pixels.
			// 949 /2 = 474, 561 - 474 + (45/2) = 109.5
			// 574 /2 = 287, 92 - 287 + (51/2) = -169.5
		case joypad.ButtonUp:
			d.direction.Draw(d.window, pixel.IM.Moved(pixel.V(-169.5, -109.5)))
		// 47, 605 and has a size of 51x45 pixels.
		// 949 /2 = 474, 605 - 474 + (45/2) = 153.5
		// 574 /2 = 287, 47 - 287 + (51/2) = -214.5
		// also rotate left 90 degrees
		case joypad.ButtonLeft:
			d.direction.Draw(d.window, pixel.IM.Rotated(pixel.ZV, math.Pi/2).Moved(pixel.V(-214.5, -153.5)))
			// 143, 605 and has a size of 51x45 pixels.
			// 949 /2 = 474, 605 - 474 + (45/2) = 153.5
			// 574 /2 = 287, 143 - 287 + (51/2) = -124.5
			// also rotate right 90 degrees
		case joypad.ButtonRight:
			d.direction.Draw(d.window, pixel.IM.Rotated(pixel.ZV, -math.Pi/2).Moved(pixel.V(-124.5, -153.5)))
			// 92, 657 and has a size of 51x45 pixels.
			// 949 /2 = 474, 657 - 474 + (45/2) = 205.5
			// 574 /2 = 287, 92 - 287 + (51/2) = -169.5
			// also rotate 180 degrees
		case joypad.ButtonDown:
			d.direction.Draw(d.window, pixel.IM.Rotated(pixel.ZV, math.Pi).Moved(pixel.V(-169.5, -205.5)))
		}

	}

	// get background colour
	// rgb := palette.GetColour(3)
	// bg := color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}
	// d.window.Clear(bg)

	// sprite := pixel.NewSprite(pixel.Picture(d.picture), pixel.R(0, 0, ppu.ScreenWidth, ppu.ScreenHeight))
	// sprite.Draw(d.window, pixel.IM)
	// d.updateCamera()
	d.window.Update()
	d.debounce.update()
}

// updateCamera updates the camera position.
func (d *Display) updateCamera() {
	xScale := d.window.Bounds().W() / ScreenWidth
	yScale := d.window.Bounds().H() / ScreenHeight
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
			d.debounce.setDebounced(button, 15)
		}
		if d.window.JustReleased(key) {
			released = append(released, button)
			d.debounce.isReleased[button] = true
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
