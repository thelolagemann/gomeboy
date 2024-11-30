package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/internal/io"
	"github.com/thelolagemann/gomeboy/pkg/utils"
	"image/color"
)

type Memory struct {
	widget.BaseWidget

	list *widget.List
	*io.Bus
}

func NewMemory(b *io.Bus) *Memory {
	m := &Memory{Bus: b}
	m.ExtendBaseWidget(m)
	return m
}

func (m *Memory) CreateRenderer() fyne.WidgetRenderer {
	var b []byte = make([]byte, 65536)
	m.Bus.CopyFrom(0, 0xffff, b)
	m.list = createHexList(b)
	return widget.NewSimpleRenderer(container.NewPadded(m.list))
}

func createHexList(data []byte) *widget.List {
	// Number of rows, each row is 16 bytes
	numRows := (len(data) + 15) / 16

	list := widget.NewList(
		// Length of the list
		func() int {
			return numRows
		},
		// Create template function
		func() fyne.CanvasObject {
			// Address, hex values (16 separate), and ASCII values in a single row
			addressLabel := mono("0x0000", themeColor(theme.ColorNameForeground))

			// Create 16 canvas.Text objects for hex values
			hexLabels := make([]fyne.CanvasObject, 16)
			for i := range hexLabels {
				hexLabels[i] = mono("00", themeColor(theme.ColorNameForeground))
			}

			// Create 16 canvas.Text objects for ASCII values
			asciiLabels := make([]fyne.CanvasObject, 16)
			for i := range asciiLabels {
				asciiLabels[i] = mono(".", themeColor(theme.ColorNameForeground))
			}

			// Combine everything into a horizontal container
			hexContainer := container.NewHBox(hexLabels...)
			asciiContainer := container.NewHBox(asciiLabels...)
			return container.NewHBox(
				addressLabel,
				mono("  ", themeColor(theme.ColorNameForeground)), // spacing
				hexContainer,
				mono("  ", themeColor(theme.ColorNameForeground)), // spacing
				asciiContainer,
			)
		},
		// Update function for a specific row
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// Get the start offset for the current row
			offset := id * 16
			address, hexValues, asciiValues := formatRow(offset, data)

			// Update the labels
			hbox := item.(*fyne.Container)

			// Update address label
			hbox.Objects[0].(*canvas.Text).Text = address

			// Update hex values (16 separate labels)
			hexContainer := hbox.Objects[2].(*fyne.Container)
			for i, hexText := range hexValues {
				hexLabel := hexContainer.Objects[i].(*canvas.Text)
				hexLabel.Text = hexText

				// Change color based on value (gray for 0x00, white otherwise)
				if hexText == "00" {
					hexLabel.Color = color.RGBA{0x7f, 0x7f, 0x7f, 255}
				} else {
					hexLabel.Color = themeColor(theme.ColorNameForeground)
				}
				hexLabel.Refresh()
			}

			// Update ASCII values (16 separate labels)
			asciiContainer := hbox.Objects[4].(*fyne.Container)
			for i, asciiText := range asciiValues {
				asciiLabel := asciiContainer.Objects[i].(*canvas.Text)
				asciiLabel.Text = asciiText

				// Optionally, you can change the color of ASCII characters here if needed
				if asciiText == "." {
					asciiLabel.Color = color.RGBA{0x7f, 0x7f, 0x7f, 255}
				} else {
					asciiLabel.Color = themeColor(theme.ColorNameForeground)
				}
				asciiLabel.Refresh()
			}

			hbox.Objects[0].Refresh()
			hbox.Objects[4].Refresh()
		},
	)
	list.HideSeparators = true

	return list
}

func (m *Memory) Refresh() {
	fmt.Println(m.list.GetScrollOffset() / 23.078125)
}

func formatRow(offset int, data []byte) (string, []string, []string) {
	// Address
	address := fmt.Sprintf("0x%04X", offset)

	// Hex values and ASCII values
	hexValues := make([]string, 16)
	asciiValues := make([]string, 16)
	for i := 0; i < 16 && offset+i < len(data); i++ {
		hexValues[i] = fmt.Sprintf("%02X", data[offset+i])
		asciiValues[i] = utils.FormatASCII(data[offset+i])
	}

	return address, hexValues, asciiValues
}
