package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/cpu"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
	"image/color"
)

type CPU struct {
	*cpu.CPU
	*io.Bus

	widget.BaseWidget

	z, n, h, c                                     *boolLabel
	regA, regF, regB, regC, regD, regE, regH, regL *registerLabel
	pc                                             *registerLabel
	sp                                             *registerLabel
	ime, halt, doubleSpeed, debug                  *boolLabel

	WindowedView
}

var (
	orange       = color.RGBA{249, 161, 72, 255}
	disabledText = color.RGBA{156, 156, 156, 255}
	disabled     = color.RGBA{35, 35, 35, 255}
	success      = color.RGBA{19, 255, 0, 255}
	errorColor   = color.RGBA{0xb0, 0x00, 0x20, 255}
	bg           = color.RGBA{47, 47, 47, 255}
	white        = color.RGBA{255, 255, 255, 255}

	registerColor   = color.RGBA{255, 255, 0, 255}
	boolColor       = color.RGBA{0, 255, 255, 255}
	boolColorNumber = color.RGBA{255, 96, 255, 255}
)

func NewCPU(cp *cpu.CPU, b *io.Bus) *CPU {
	c := &CPU{
		Bus: b,
		CPU: cp,

		z:           newBoolLabel("Z", true),
		n:           newBoolLabel("N", true),
		h:           newBoolLabel("H", true),
		c:           newBoolLabel("C", true),
		regA:        newRegisterLabel("A", registerColor, false),
		regF:        newRegisterLabel("F", registerColor, false),
		regB:        newRegisterLabel("B", registerColor, false),
		regC:        newRegisterLabel("C", registerColor, false),
		regD:        newRegisterLabel("D", registerColor, false),
		regE:        newRegisterLabel("E", registerColor, false),
		regH:        newRegisterLabel("H", registerColor, false),
		regL:        newRegisterLabel("L", registerColor, false),
		pc:          newRegisterLabel("PC", orange, true),
		sp:          newRegisterLabel("SP", orange, true),
		ime:         newBoolLabel("IME", false),
		halt:        newBoolLabel("HALT", false),
		doubleSpeed: newBoolLabel("DOUBLE SPEED", false),
		debug:       newBoolLabel("DEBUG", false),
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *CPU) AttachWindow(w fyne.Window) { c.Window = w }

func (c *CPU) CreateRenderer() fyne.WidgetRenderer {
	c.Window.SetFixedSize(true)
	flags := container.NewGridWithColumns(2, c.z, c.n, c.h, c.c)
	grid := container.NewGridWithColumns(2,
		c.regA, c.regF,
		c.regB, c.regC,
		c.regD, c.regE,
		c.regH, c.regL,
	)

	return widget.NewSimpleRenderer(container.NewVBox(flags, grid, c.pc, c.sp, c.ime, c.halt, c.doubleSpeed, c.debug))
}

// Refresh updates the values of the CPU registers in the widget
func (c *CPU) Refresh() {
	c.z.setValue(c.F&types.Bit7 > 0)
	c.n.setValue(c.F&types.Bit6 > 0)
	c.h.setValue(c.F&types.Bit5 > 0)
	c.c.setValue(c.F&types.Bit4 > 0)
	c.regA.setValue(c.A)
	c.regF.setValue(c.F)
	c.regB.setValue(c.B)
	c.regC.setValue(c.C)
	c.regD.setValue(c.D)
	c.regE.setValue(c.E)
	c.regH.setValue(c.H)
	c.regL.setValue(c.L)
	c.pc.setValue(c.PC)
	c.sp.setValue(c.SP)
	c.ime.setValue(c.InterruptsEnabled())
	c.halt.setValue(c.Halted)
	c.doubleSpeed.setValue(c.DoubleSpeed)

	c.debug.setValue(c.Debug)
}

type registerLabel struct {
	widget.BaseWidget

	name      string
	nameColor color.Color
	wide      bool

	hexValue, binaryValue0, binaryValue1 *canvas.Text
}

func newRegisterLabel(name string, c color.Color, wide bool) *registerLabel {
	l := &registerLabel{name: name, nameColor: c, wide: wide}
	l.ExtendBaseWidget(l)
	return l
}

func (l *registerLabel) CreateRenderer() fyne.WidgetRenderer {
	name := canvas.NewText(l.name, l.nameColor)
	name.TextStyle.Monospace = true
	name.TextSize = 14

	l.hexValue = canvas.NewText("0x00", white)
	l.hexValue.TextStyle.Monospace = true

	l.binaryValue0 = canvas.NewText("0000 0000", white)
	l.binaryValue0.TextStyle.Monospace = true

	background := canvas.NewRectangle(bg)
	background.CornerRadius = 5

	var binaryContainer *fyne.Container
	if l.wide {
		l.binaryValue1 = canvas.NewText("0000 0000", white)
		l.binaryValue1.TextStyle.Monospace = true

		binaryContainer = container.NewBorder(nil, nil, l.binaryValue0, l.binaryValue1)
	} else {
		binaryContainer = container.NewHBox(l.binaryValue0)
	}

	registerContainer := container.NewVBox(container.NewBorder(nil, nil, name, l.hexValue), binaryContainer)
	content := container.NewStack(background, container.NewPadded(registerContainer))

	background.Resize(content.MinSize())

	return widget.NewSimpleRenderer(content)
}

func (l *registerLabel) setValue(v interface{}) {
	zero := false
	switch value := v.(type) {
	case uint8:
		zero = value == 0
		if l.hexValue.Text == fmt.Sprintf("= 0x%02X", value) {
			return // value unchanged
		}
		l.hexValue.Text = fmt.Sprintf("= 0x%02X", value)
		binaryText := fmt.Sprintf("%08b", value)
		l.binaryValue0.Text = binaryText[:4] + " " + binaryText[4:]
	case uint16:
		zero = value == 0
		if l.hexValue.Text == fmt.Sprintf("= 0x%04X", value) {
			return
		}
		l.hexValue.Text = fmt.Sprintf("= 0x%04X", value)
		binaryText := fmt.Sprintf("%016b", value)
		l.binaryValue0.Text = binaryText[:4] + " " + binaryText[4:8]
		l.binaryValue1.Text = binaryText[8:12] + " " + binaryText[12:]
	}

	var targetColor = white
	if zero {
		targetColor = disabledText
	}
	currentColor := interpolateColor(l.binaryValue0.Color.(color.RGBA), targetColor, 0.1)
	l.hexValue.Color = currentColor
	l.hexValue.Refresh()

	l.binaryValue0.Color = currentColor
	l.binaryValue0.Refresh()

	if l.wide {
		l.binaryValue1.Color = currentColor
		l.binaryValue1.Refresh()
	}
}

type boolLabel struct {
	widget.BaseWidget

	name     string
	numbered bool
	value    *canvas.Text
}

func newBoolLabel(name string, numbered bool) *boolLabel {
	l := &boolLabel{name: name, numbered: numbered}
	l.ExtendBaseWidget(l)
	return l
}

func (l *boolLabel) CreateRenderer() fyne.WidgetRenderer {
	name := canvas.NewText(l.name, boolColor)
	if l.numbered {
		name.Color = boolColorNumber
	}
	name.TextStyle.Monospace = true
	name.TextSize = 14

	l.value = canvas.NewText("  ON", success)
	l.value.TextStyle.Monospace = true

	valueBackground := canvas.NewRectangle(color.RGBA{65, 65, 65, 255})
	valueBackground.CornerRadius = 3

	valueContainer := container.NewStack(valueBackground, container.NewPadded(l.value))

	content := container.NewVBox(container.NewBorder(nil, nil, container.NewPadded(name), valueContainer))

	background := canvas.NewRectangle(bg)
	background.CornerRadius = 5

	finalContent := container.NewStack(background, content)

	background.Resize(finalContent.MinSize())

	return widget.NewSimpleRenderer(finalContent)
}

func (l *boolLabel) setValue(v bool) {
	var valueText string
	var valueColor color.RGBA
	if l.numbered {
		valueText = " 1"
		valueColor = white
		if !v {
			valueText = " 0"
			valueColor = disabledText
		}
	} else {
		valueText = "  ON"
		valueColor = success
		if !v {
			valueText = " OFF"
			valueColor = disabledText
		}
	}

	l.value.Text = valueText
	l.value.Color = interpolateColor(l.value.Color.(color.RGBA), valueColor, 0.1)
	l.value.Refresh()
}

func interpolateColor(c1, c2 color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c1.R) + t*(float64(c2.R)-float64(c1.R))),
		G: uint8(float64(c1.G) + t*(float64(c2.G)-float64(c1.G))),
		B: uint8(float64(c1.B) + t*(float64(c2.B)-float64(c1.B))),
		A: uint8(float64(c1.A) + t*(float64(c2.A)-float64(c1.A))),
	}
}
