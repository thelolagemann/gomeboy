package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/cpu"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/internal/types"
	"github.com/thelolagemann/gomeboy/pkg/display/fyne/themes"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"image/color"
	"math"
)

type CPU struct {
	widget.BaseWidget

	*cpu.CPU
	*io.Bus

	z, n, h, c                                     *boolLabel
	regA, regF, regB, regC, regD, regE, regH, regL *registerLabel
	pc                                             *registerLabel
	sp                                             *registerLabel
	ime, halt, doubleSpeed, debug                  *boolLabel
}

func NewCPU(cp *cpu.CPU, b *io.Bus) *CPU {
	c := &CPU{
		Bus: b,
		CPU: cp,

		z:           newBoolLabel("Z", true),
		n:           newBoolLabel("N", true),
		h:           newBoolLabel("H", true),
		c:           newBoolLabel("C", true),
		regA:        newRegisterLabel("A", themeColor(themes.ColorNameSecondary), false),
		regF:        newRegisterLabel("F", themeColor(themes.ColorNameSecondary), false),
		regB:        newRegisterLabel("B", themeColor(themes.ColorNameSecondary), false),
		regC:        newRegisterLabel("C", themeColor(themes.ColorNameSecondary), false),
		regD:        newRegisterLabel("D", themeColor(themes.ColorNameSecondary), false),
		regE:        newRegisterLabel("E", themeColor(themes.ColorNameSecondary), false),
		regH:        newRegisterLabel("H", themeColor(themes.ColorNameSecondary), false),
		regL:        newRegisterLabel("L", themeColor(themes.ColorNameSecondary), false),
		pc:          newRegisterLabel("PC", themeColor(theme.ColorNamePrimary), true),
		sp:          newRegisterLabel("SP", themeColor(theme.ColorNamePrimary), true),
		ime:         newBoolLabel("IME", false),
		halt:        newBoolLabel("HALT", false),
		doubleSpeed: newBoolLabel("DOUBLE SPEED", false),
		debug:       newBoolLabel("DEBUG", false),
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *CPU) CreateRenderer() fyne.WidgetRenderer {
	findWindow("CPU").SetFixedSize(true)
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
	name := mono(l.name, l.nameColor)
	l.hexValue = mono("0x00", themeColor(theme.ColorNameForeground))
	l.binaryValue0 = mono("0000 0000", themeColor(theme.ColorNameForeground))

	var binaryContainer *fyne.Container
	if l.wide {
		l.binaryValue1 = mono("0000 0000", themeColor(theme.ColorNameForeground))
		binaryContainer = container.NewBorder(nil, nil, l.binaryValue0, l.binaryValue1)
	} else {
		binaryContainer = container.NewHBox(l.binaryValue0)
	}

	registerContainer := container.NewVBox(container.NewBorder(nil, nil, name, l.hexValue), binaryContainer)
	return widget.NewSimpleRenderer(newBadge(themeColor(themes.ColorNameBackgroundOnBackground), 5, container.NewPadded(registerContainer)))
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

	var targetColor = themeColor(theme.ColorNameForeground)
	if zero {
		targetColor = themeColor(themes.ColorNameDisabledText)
	}
	currentColor := interpolateColor(l.binaryValue0.Color, targetColor, 0.4)
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
	name := mono(l.name, themeColor(themes.ColorNameBool))
	if l.numbered {
		name.Color = themeColor(themes.ColorNameBoolNumber)
	}

	l.value = mono("  ON", themeColor(theme.ColorNameSuccess))
	valueContainer := newBadge(themeColor(theme.ColorNameButton), 3, container.NewPadded(l.value))
	content := container.NewVBox(container.NewBorder(nil, nil, container.NewPadded(name), valueContainer))

	return widget.NewSimpleRenderer(newBadge(themeColor(themes.ColorNameBackgroundOnBackground), 5, content))
}

func (l *boolLabel) setValue(v bool) {
	var valueText = " 1"
	var valueColor = themeColor(theme.ColorNameForeground)
	switch {
	case l.numbered && !v:
		valueText = " 0"
		valueColor = themeColor(themes.ColorNameDisabledText)
	case !l.numbered && v:
		valueText = "  ON"
		valueColor = themeColor(theme.ColorNameSuccess)
	case !l.numbered && !v:
		valueText = " OFF"
		valueColor = themeColor(themes.ColorNameDisabledText)
	}
	if l.value.Text == valueText && l.value.Color == valueColor {
		return
	}

	l.value.Text = valueText
	l.value.Color = interpolateColor(l.value.Color, valueColor, 0.4)
	l.value.Refresh()
}

func interpolateColor(col1, col2 color.Color, t float64) color.Color {
	t = utils.Clamp(0, t, 1)

	// Convert both colors to RGBA values
	r1, g1, b1, a1 := col1.RGBA()
	r2, g2, b2, a2 := col2.RGBA()

	// Normalize the RGBA values to the range [0, 255]
	r1f := float64(r1) / 257.0
	g1f := float64(g1) / 257.0
	b1f := float64(b1) / 257.0
	a1f := float64(a1) / 257.0

	r2f := float64(r2) / 257.0
	g2f := float64(g2) / 257.0
	b2f := float64(b2) / 257.0
	a2f := float64(a2) / 257.0

	// Perform linear interpolation on each channel
	interpolatedR := (1-t)*r1f + t*r2f
	interpolatedG := (1-t)*g1f + t*g2f
	interpolatedB := (1-t)*b1f + t*b2f
	interpolatedA := (1-t)*a1f + t*a2f

	// Convert the interpolated values back to uint8 (clamping to avoid rounding issues)
	return color.NRGBA{
		R: uint8(math.Round(math.Min(r2f, math.Max(0, interpolatedR)))),
		G: uint8(math.Round(math.Min(g2f, math.Max(0, interpolatedG)))),
		B: uint8(math.Round(math.Min(b2f, math.Max(0, interpolatedB)))),
		A: uint8(math.Round(math.Min(a2f, math.Max(0, interpolatedA)))),
	}
}
