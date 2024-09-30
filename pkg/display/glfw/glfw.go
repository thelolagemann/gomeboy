//go:build !android

package glfw

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/emulator"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"github.com/veandco/go-sdl2/sdl"
	"runtime"
	"time"
)

const (
	aspectRatio = float32(160) / float32(144)
)

func init() {
	// GLFW: this is needed to arrange for main to run on main thread
	runtime.LockOSThread()

	// initialize GLFW
	if err := glfw.Init(); err != nil {
		log.Fatal(err.Error())
	}

	// initialize OpenGL
	if err := gl.Init(); err != nil {
		log.Fatal(err.Error())
	}

	mon = glfw.GetPrimaryMonitor()

	// register display driver
	driver := &glfwDriver{}
	display.Install("glfw", driver, []display.DriverOption{
		{
			Name:        "fullscreen",
			Default:     false,
			Value:       &driver.fullscreen,
			Type:        "bool",
			Description: "Run in fullscreen mode",
		},
		{
			Name:        "scale",
			Default:     4.0,
			Value:       &driver.scale,
			Type:        "float",
			Description: "Scale the window by this factor",
		},
		{
			Name:        "maintain-aspect-ratio",
			Default:     false,
			Value:       &driver.maintainAspectRatio,
			Type:        "bool",
			Description: "Force the window to maintain the correct aspect ratio",
		},
	})

	if err := sdl.Init(sdl.INIT_JOYSTICK | sdl.INIT_VIDEO); err != nil {
		panic(err)
	}

	if sdl.NumJoysticks() > 0 {
		sdl.JoystickEventState(sdl.ENABLE)
		for i := 0; i < sdl.NumJoysticks(); i++ {
			joystick = sdl.JoystickOpen(i)
		}
	}
}

var (
	joypadKeys = map[glfw.Key]io.Button{
		glfw.KeyA:         io.ButtonA,
		glfw.KeyB:         io.ButtonB,
		glfw.KeyDown:      io.ButtonDown,
		glfw.KeyUp:        io.ButtonUp,
		glfw.KeyLeft:      io.ButtonLeft,
		glfw.KeyRight:     io.ButtonRight,
		glfw.KeyEnter:     io.ButtonStart,
		glfw.KeyBackspace: io.ButtonSelect,
	}
)

var (
	mon      *glfw.Monitor
	joystick *sdl.Joystick
)

// glfwDriver implements a barebones display driver using GLFW
// and the OpenGL API.
type glfwDriver struct {
	fullscreen          bool
	scale               float64
	maintainAspectRatio bool

	accelerometerX, accelerometerY float32
	windowSettings                 struct {
		width      int
		height     int
		xPos, yPos int
	}
}

// Start starts the display driver.
func (g *glfwDriver) Start(c emulator.Controller, frames <-chan []byte, pressed, released chan<- io.Button) error {
	// create window
	window, err := glfw.CreateWindow(int(160*g.scale), int(144*g.scale), "GomeBoy", nil, nil)
	if err != nil {
		return err
	}

	if g.maintainAspectRatio {
		window.SetAspectRatio(10, 9)
	}
	// fullscreen
	if g.fullscreen {
		bestMode := getBestMode()
		window.SetMonitor(mon, 0, 0, bestMode.Width, bestMode.Height, bestMode.RefreshRate)
	}

	window.MakeContextCurrent()

	// initialize window settings
	g.windowSettings.width, g.windowSettings.height = window.GetSize()
	g.windowSettings.xPos, g.windowSettings.yPos = window.GetPos()

	var texture uint32
	{
		gl.GenTextures(1, &texture)

		gl.BindTexture(gl.TEXTURE_2D, texture)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

		gl.BindImageTexture(0, texture, 0, false, 0, gl.WRITE_ONLY, gl.RGB8)
	}

	// setup event handling
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// check to see if the key is mapped to a joypad button
		if button, ok := joypadKeys[key]; ok {
			switch action {
			case glfw.Press:
				pressed <- button
			case glfw.Release:
				released <- button
			}
		}

		if action == glfw.Press {
			switch key {
			case glfw.KeyF11:
				// toggle fullscreen
				if g.fullscreen {
					window.SetMonitor(nil, g.windowSettings.xPos, g.windowSettings.yPos, g.windowSettings.width, g.windowSettings.height, 60)
				} else {
					// store the current window settings
					g.windowSettings.width, g.windowSettings.height = window.GetSize()
					g.windowSettings.xPos, g.windowSettings.yPos = window.GetPos()

					bestMode := getBestMode()
					window.SetMonitor(mon, 0, 0, bestMode.Width, bestMode.Height, bestMode.RefreshRate)
				}

				g.fullscreen = !g.fullscreen
			case glfw.KeyEscape, glfw.KeyPause:
				if c.Paused() {
					c.Resume()
				} else {
					c.Pause()
				}
			}
		}
	})

	var fb uint32
	{
		gl.GenFramebuffers(1, &fb)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fb)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0)

		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fb)
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	}

	// handle resizing
	targetWidth := int32(160 * g.scale)
	targetHeight := int32(144 * g.scale)
	var offsetX, offsetY int32
	window.SetSizeCallback(func(_ *glfw.Window, w, h int) {

		if float32(w)/float32(h) > aspectRatio {
			targetWidth = int32(float32(h) * aspectRatio)
			targetHeight = int32(h)
		} else {
			targetWidth = int32(w)
			targetHeight = int32(float32(w) / aspectRatio)
		}

		offsetX = (int32(w) - targetWidth) / 2
		offsetY = (int32(h) - targetHeight) / 2
	})

	pollTicker := time.NewTicker(time.Millisecond * 20) // to handle when paused
	var sdlEvent sdl.Event
	// draw loop
	for {
		select {
		case f := <-frames:
			glfw.PollEvents()
			if window.ShouldClose() {
				return nil
			}
			gl.Clear(gl.COLOR_BUFFER_BIT)

			gl.BindTexture(gl.TEXTURE_2D, texture)
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB8, 160, 144, 0, gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(f))

			gl.BlitFramebuffer(0, 0, 160, 144, offsetX, offsetY+targetHeight, offsetX+targetWidth, offsetY, gl.COLOR_BUFFER_BIT, gl.NEAREST)

			window.SwapBuffers()
		case <-pollTicker.C:

			for sdlEvent = sdl.PollEvent(); sdlEvent != nil; sdlEvent = sdl.PollEvent() {
				switch t := sdlEvent.(type) {
				case *sdl.JoyAxisEvent:
					switch t.Axis {
					case 3: // x-axis
						g.accelerometerX = -(float32(t.Value) / 32768.0)
					case 4: // y-axis
						g.accelerometerY = -(float32(t.Value) / 32768.0)
					}
				}
			}

			glfw.PollEvents()
		}
	}

}

// Stop stops the display driver.
func (g *glfwDriver) Stop() error {
	glfw.Terminate()

	return nil
}

func (g *glfwDriver) X() float32 {
	return g.accelerometerX
}

func (g *glfwDriver) Y() float32 {
	return g.accelerometerY
}

func (g *glfwDriver) Rumble(lowFreq, highFreq uint16, duration uint32) {
	joystick.Rumble(lowFreq, highFreq, duration)
}

// getBestMode returns the best video mode for the current monitor
// by choosing the highest resolution that is the closest match to
// the native aspect ratio of the monitor. This should provide a
// reasonable default for most monitors.
func getBestMode() *glfw.VidMode {
	sizeX, sizeY := mon.GetPhysicalSize()
	monAspectRatio := float32(sizeX) / float32(sizeY)
	closestMatch := float32(0)

	modes := mon.GetVideoModes()
	var best = modes[len(modes)-1]
	for _, vm := range modes {
		// skip modes that aren't at least 60FPS
		if vm.RefreshRate < 60 {
			continue
		}

		// skip modes that have a worse aspect ratio match
		vmAspectRatio := float32(vm.Width) / float32(vm.Height)
		if monAspectRatio-vmAspectRatio > closestMatch {
			continue
		}

		closestMatch = vmAspectRatio - monAspectRatio
		best = vm
	}

	return best
}
