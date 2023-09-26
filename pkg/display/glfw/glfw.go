package glfw

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/thelolagemann/gomeboy/internal/gameboy"
	"github.com/thelolagemann/gomeboy/internal/joypad"
	"github.com/thelolagemann/gomeboy/pkg/display"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/thelolagemann/gomeboy/pkg/log"
	"runtime"
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

	display.Install("glfw", &glfwDriver{})
}

var (
	joypadKeys = map[glfw.Key]joypad.Button{
		glfw.KeyA:      joypad.ButtonA,
		glfw.KeyB:      joypad.ButtonB,
		glfw.KeyDown:   joypad.ButtonDown,
		glfw.KeyUp:     joypad.ButtonUp,
		glfw.KeyLeft:   joypad.ButtonLeft,
		glfw.KeyRight:  joypad.ButtonRight,
		glfw.KeyEnter:  joypad.ButtonStart,
		glfw.KeyEscape: joypad.ButtonSelect,
	}
)

// glfwDriver implements a barebones display driver using GLFW
// and the OpenGL API.
type glfwDriver struct {
}

func (g *glfwDriver) Attach(gb *gameboy.GameBoy) {
	// no-op for glfw
}

// Start starts the display driver.
func (g *glfwDriver) Start(frames <-chan []byte, evts <-chan event.Event, pressed, released chan<- joypad.Button) error {
	// create window
	window, err := glfw.CreateWindow(160, 144, "GomeBoy", nil, nil)
	if err != nil {
		return err
	}

	window.MakeContextCurrent()

	var texture uint32
	{
		gl.GenTextures(1, &texture)

		gl.BindTexture(gl.TEXTURE_2D, texture)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
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
	})

	var fb uint32
	{
		gl.GenFramebuffers(1, &fb)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fb)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture, 0)

		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fb)
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	}

	// draw loop
	for {
		select {
		case f := <-frames:
			if window.ShouldClose() {
				return nil
			}
			var w, h = window.GetSize()

			gl.BindTexture(gl.TEXTURE_2D, texture)
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB8, 160, 144, 0, gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(f))

			gl.BlitFramebuffer(0, 0, 160, 144, 0, int32(h), int32(w), 0, gl.COLOR_BUFFER_BIT, gl.NEAREST)

			window.SwapBuffers()
			glfw.PollEvents()
		case e := <-evts:
			switch e.Type {
			case event.Title:
				window.SetTitle(e.Data.(string))
			}
		}
	}

}

// Stop stops the display driver.
func (g *glfwDriver) Stop() error {
	glfw.Terminate()

	return nil
}
