package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"image/color"
)

var (
	_ display.View = (*CPU)(nil)
)

type CPU struct {
	*cpu.CPU

	flags []binding.BoolList
}

func (c *CPU) Title() string {
	return "CPU"
}

func (c *CPU) Run(window fyne.Window, events <-chan display.Event) error {
	grid := container.NewVBox()
	// set the content of the window
	window.SetContent(grid)

	// create the labels
	af := canvas.NewText("", color.White)
	af.TextStyle = fyne.TextStyle{Monospace: true}
	bc := canvas.NewText("", color.White)
	bc.TextStyle = fyne.TextStyle{Monospace: true}
	de := canvas.NewText("", color.White)
	de.TextStyle = fyne.TextStyle{Monospace: true}
	hl := canvas.NewText("", color.White)
	hl.TextStyle = fyne.TextStyle{Monospace: true}
	sp := canvas.NewText("", color.White)
	sp.TextStyle = fyne.TextStyle{Monospace: true}
	pc := canvas.NewText("", color.White)
	pc.TextStyle = fyne.TextStyle{Monospace: true}

	// create a grid for the registers
	registerGrid := container.NewGridWithRows(7)
	registerGrid.Add(container.NewGridWithColumns(2, widget.NewLabel("AF"), af))
	registerGrid.Add(container.NewGridWithColumns(2, widget.NewLabel("BC"), bc))
	registerGrid.Add(container.NewGridWithColumns(2, widget.NewLabel("DE"), de))
	registerGrid.Add(container.NewGridWithColumns(2, widget.NewLabel("HL"), hl))
	registerGrid.Add(container.NewGridWithColumns(2, widget.NewLabel("SP"), sp))
	registerGrid.Add(container.NewGridWithColumns(2, widget.NewLabel("PC"), pc))

	// add the grid to the window
	grid.Add(registerGrid)

	// handle events
	go func() {
		for {
			select {
			case e := <-events:
				switch e.Type {
				case display.EventTypeQuit:
					return
				case display.EventTypeFrame:
					registers := e.State.CPU.Registers

					// update bindings
					af.Text = fmt.Sprintf("%04X", registers.AF)
					bc.Text = fmt.Sprintf("%04X", registers.BC)
					de.Text = fmt.Sprintf("%04X", registers.DE)
					hl.Text = fmt.Sprintf("%04X", registers.HL)
					sp.Text = fmt.Sprintf("%04X", registers.SP)
					pc.Text = fmt.Sprintf("%04X", registers.PC)

					registerGrid.Refresh()
				}
			}
		}
	}()

	return nil
}

func (c *CPU) Setup(w fyne.Window) error {

	// create a grid for register F flags (checkboxes for each flag of the 4 flags)
	fGrid := container.NewVBox(widget.NewLabel("F Flags:"))

	// create bool bindings for each flag
	zBool := binding.NewBool()
	nBool := binding.NewBool()
	hBool := binding.NewBool()
	cyBool := binding.NewBool()

	// create a checkbox for each flag
	z := widget.NewCheck("Z", func(b bool) {
		// do nothing
	})
	n := widget.NewCheck("N", func(b bool) {
		// do nothing
	})
	h := widget.NewCheck("H", func(b bool) {
		// do nothing
	})
	cy := widget.NewCheck("CY", func(b bool) {
		// do nothing
	})

	// bind the checkboxes to the bool bindings
	z.Bind(zBool)
	n.Bind(nBool)
	h.Bind(hBool)
	cy.Bind(cyBool)

	// disable the checkboxes
	z.Disable()
	n.Disable()
	h.Disable()
	cy.Disable()

	// add the checkboxes to the grid
	fGrid.Add(z)
	fGrid.Add(n)
	fGrid.Add(h)
	fGrid.Add(cy)

	// Interrupts

	// create divider

	// create a grid for the interrupts
	interruptsGrid := container.NewVBox(widget.NewLabel("Interrupts:"))

	// create bool bindings for each interrupt
	vBlankBool := binding.NewBool()
	lcdStatBool := binding.NewBool()
	timerBool := binding.NewBool()
	serialBool := binding.NewBool()
	joypadBool := binding.NewBool()

	// create a checkbox for each interrupt
	vBlank := widget.NewCheck("VBlank", func(b bool) {
		// do nothing
	})
	lcdStat := widget.NewCheck("LCDStat", func(b bool) {
		// do nothing
	})
	timer := widget.NewCheck("Timer", func(b bool) {
		// do nothing
	})
	serial := widget.NewCheck("Serial", func(b bool) {
		// do nothing
	})
	joypad := widget.NewCheck("Joypad", func(b bool) {
		// do nothing
	})

	// bind the checkboxes to the bool bindings
	vBlank.Bind(vBlankBool)
	lcdStat.Bind(lcdStatBool)
	timer.Bind(timerBool)
	serial.Bind(serialBool)
	joypad.Bind(joypadBool)

	// add the checkboxes to the grid
	interruptsGrid.Add(vBlank)
	interruptsGrid.Add(lcdStat)
	interruptsGrid.Add(timer)
	interruptsGrid.Add(serial)
	interruptsGrid.Add(joypad)

	return nil
}

func NewCPU(c *cpu.CPU) *CPU {
	return &CPU{CPU: c}
}
