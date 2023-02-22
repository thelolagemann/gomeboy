package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/cpu"
	"github.com/thelolagemann/go-gameboy/pkg/display"
)

var (
	_ display.View = (*CPU)(nil)
)

type CPU struct {
	*cpu.CPU

	registerGrid *widget.TextGrid
	grid         *fyne.Container
	flags        []binding.BoolList
}

func (c *CPU) Run(events <-chan display.Event) error {
	for {
		select {
		case e := <-events:
			switch e.Type {
			case display.EventTypeQuit:
				return nil
			case display.EventTypeFrame:
				// TODO get cpu values from event data and update the text grid
				c.registerGrid.SetText(fmt.Sprintf("Registers:\n\nPC\t%04X\nSP\t%04X\nAF\t%04X\nBC\t%04X\nDE\t%04X\nHL\t%04X\n", c.PC, c.SP, c.AF.Uint16(), c.BC.Uint16(), c.DE.Uint16(), c.HL.Uint16()))

				c.grid.Refresh()
			}
		}
	}
}

func (c *CPU) Setup(w fyne.Window) error {
	// create the base grid
	c.grid = container.NewGridWithColumns(2)

	// create a new text grid
	c.registerGrid = widget.NewTextGridFromString("Registers:\n\nPC\t0000\nSP\t0000\nAF\t0000\nBC\t0000\nDE\t0000\nHL\t0000\n")

	// add the text grid to the grid
	c.grid.Add(c.registerGrid)

	// set the content of the window to the grid
	w.SetContent(c.grid)

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

	// add the grid to the main grid
	c.grid.Add(fGrid)

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

	// add the divider and grid to the main grid
	c.grid.Add(interruptsGrid)

	return nil
}

func NewCPU(c *cpu.CPU) *CPU {
	return &CPU{CPU: c}
}
