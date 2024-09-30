package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/cpu"
	"github.com/thelolagemann/gomeboy/pkg/utils"
)

type CPU struct {
	*cpu.CPU

	widget.BaseWidget

	regA *widget.Label
	regB *widget.Label
	regC *widget.Label
	regD *widget.Label
	regE *widget.Label
	regH *widget.Label
	regL *widget.Label
	regF *widget.Label
	pc   *widget.Label
	sp   *widget.Label
}

func NewCPU(cp *cpu.CPU) *CPU {
	c := &CPU{
		CPU:  cp,
		regA: widget.NewLabel("0x00"),
		regB: widget.NewLabel("0x00"),
		regC: widget.NewLabel("0x00"),
		regD: widget.NewLabel("0x00"),
		regE: widget.NewLabel("0x00"),
		regH: widget.NewLabel("0x00"),
		regL: widget.NewLabel("0x00"),
		regF: widget.NewLabel("Z0 N0 H0 C0"),
		pc:   widget.NewLabel("0x0000"),
		sp:   widget.NewLabel("0x0000"),
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *CPU) CreateRenderer() fyne.WidgetRenderer {
	grid := container.NewGridWithColumns(2,
		widget.NewLabel("A:"), c.regA,
		widget.NewLabel("B:"), c.regB,
		widget.NewLabel("C:"), c.regC,
		widget.NewLabel("D:"), c.regD,
		widget.NewLabel("E:"), c.regE,
		widget.NewLabel("H:"), c.regH,
		widget.NewLabel("L:"), c.regL,
		widget.NewLabel("PC:"), c.pc,
		widget.NewLabel("SP:"), c.sp,
		widget.NewLabel("Flags:"), c.regF,
	)
	for _, l := range grid.Objects {
		if label, ok := l.(*widget.Label); ok {
			label.TextStyle.Monospace = true
		}
	}
	return widget.NewSimpleRenderer(grid)
}

// Refresh updates the values of the CPU registers in the widget
func (c *CPU) Refresh() {
	c.regA.SetText(fmt.Sprintf("0x%02X", c.A))
	c.regB.SetText(fmt.Sprintf("0x%02X", c.B))
	c.regC.SetText(fmt.Sprintf("0x%02X", c.C))
	c.regD.SetText(fmt.Sprintf("0x%02X", c.D))
	c.regE.SetText(fmt.Sprintf("0x%02X", c.E))
	c.regH.SetText(fmt.Sprintf("0x%02X", c.H))
	c.regL.SetText(fmt.Sprintf("0x%02X", c.L))
	c.pc.SetText(fmt.Sprintf("0x%04X", c.PC))
	c.sp.SetText(fmt.Sprintf("0x%04X", c.SP))
	c.regF.SetText(fmt.Sprintf("Z%s N%s H%s C%s", utils.BoolToString(c.F&0x80 > 0), utils.BoolToString(c.F&0x40 > 0), utils.BoolToString(c.F&0x20 > 0), utils.BoolToString(c.F&0x10 > 0)))
}
