package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/go-gameboy/internal/cheats"
	"github.com/thelolagemann/go-gameboy/pkg/display"
	"strings"
	"unicode"
)

type cheatEntryLayout struct {
}

func (c *cheatEntryLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(300, 35)
	}
	// get the base text box to see if there is a subheading or not
	base := objects[0].(*fyne.Container)
	if len(base.Objects) == 2 {
		return fyne.NewSize(300, 65)
	}
	return fyne.NewSize(300, 35)
}

func (c *cheatEntryLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}

	// get the objects
	code := objects[0].(*fyne.Container).Objects[0].(*widget.Label)

	if len(objects) == 3 {
		desc := objects[0].(*fyne.Container).Objects[1].(*widget.Label)
		desc.Move(fyne.NewPos(100, 0))
	}
	enabled := objects[1].(*widget.Check)

	// set the position and size of the objects
	code.Move(fyne.NewPos(0, 0))

	// enabled should be right aligned
	enabled.Move(fyne.NewPos(size.Width-(enabled.Size().Width*1.5), (size.Height/2)-(enabled.Size().Height/2)))
	enabled.Resize(fyne.NewSize(20, 20))
}

type titleLayout struct {
}

func (t titleLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(300, 30)
}

func (t titleLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}

	// get the objects
	title := objects[0].(*widget.Label)

	// set the position and size of the objects
	title.Move(fyne.NewPos(0, 0))

	// checkbox should be right aligned
	checkbox := objects[1].(*widget.Check)
	checkbox.Move(fyne.NewPos(size.Width-(checkbox.Size().Width*1.5), (size.Height/2)-(checkbox.Size().Height/2)))
	checkbox.Resize(fyne.NewSize(20, 20))
}

type CheatManager struct {
	genie *cheats.GameGenie
	shark *cheats.GameShark

	sharkList       *fyne.Container
	genieList       *fyne.Container
	sharkSubheading bool
	genieSubheading bool
}

func (cm *CheatManager) LoadedCheatCount() int {
	var total = 0
	if cm.genie != nil {
		total += len(cm.genie.Codes)
	}

	if cm.shark != nil {
		total += len(cm.shark.Codes)
	}

	return total
}

func NewCheatManager(opts ...CheatManagerOption) *CheatManager {
	cm := &CheatManager{}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

type CheatManagerOption func(*CheatManager)

func WithGameGenie(genie *cheats.GameGenie) CheatManagerOption {
	return func(cm *CheatManager) {
		cm.genie = genie
	}
}

func WithGameShark(shark *cheats.GameShark) CheatManagerOption {
	return func(cm *CheatManager) {
		cm.shark = shark
	}
}

func (cm *CheatManager) Run(window fyne.Window, events <-chan display.Event) error {
	// create a grid for the lists
	cheatGrid := container.NewVBox()

	// set the content of the window
	window.SetContent(cheatGrid)
	//window.Resize(fyne.NewSize(300, 700))

	// create a list for the game genie cheats if enabled
	if cm.genie != nil {
		// create a grid for the game genie
		gameGenieGrid := container.NewVBox()

		// create the title box with title and debug select
		gameGenieTitleBox := container.New(titleLayout{})
		gameGenieTitle := widget.NewLabel("Game Genie")
		gameGenieTitleBox.Add(gameGenieTitle)

		// create the debug checkbox
		debugCheckbox := widget.NewCheck("", func(checked bool) {
			cm.genieSubheading = checked
			cm.refreshGenieCheats()
		})

		gameGenieTitleBox.Add(debugCheckbox)

		// add the title box to the genie grid
		gameGenieGrid.Add(gameGenieTitleBox)

		// create a grid for the input
		gameGenieInputGrid := container.NewGridWithRows(3)

		// create the input box
		gameGenieInput := widget.NewEntry()
		gameGenieInput.SetPlaceHolder("Enter Game Genie Code")

		var lastLen = 0
		gameGenieInput.OnChanged = func(text string) {
			textLen := len(text)
			if lastLen > textLen {
				lastLen = textLen
				return
			}
			if textLen > 11 {
				gameGenieInput.Text = text[:11]
				gameGenieInput.Refresh()
				return
			}

			wasInvalid := false
			// remove any non alphanumeric characters
			gameGenieInput.Text = strings.Map(func(r rune) rune {
				if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' {
					return r
				}
				wasInvalid = true
				return -1
			}, text)

			if wasInvalid {
				gameGenieInput.Refresh()
				return
			}

			// TODO prevent user input hyphen, limit input to 9 characters
			if textLen == 3 || textLen == 7 {
				gameGenieInput.Text += "-"
				gameGenieInput.CursorColumn++

				gameGenieInput.Refresh()
			}

			if (textLen == 4 || textLen == 8) && text[textLen-1] != '-' {
				gameGenieInput.Text = text[:textLen-1] + "-" + text[textLen-1:]
				gameGenieInput.CursorColumn++

				gameGenieInput.Refresh()
			}

			if (textLen == 5 || textLen == 9) && (text[textLen-1] == '-' && text[textLen-2] == '-') {
				gameGenieInput.Text = text[:textLen-2] + text[textLen-1:]
				gameGenieInput.CursorColumn--

				gameGenieInput.Refresh()
			}

			lastLen = len(gameGenieInput.Text)
		}
		gameGenieNameInput := widget.NewEntry()
		gameGenieNameInput.SetPlaceHolder("Enter Cheat Name")
		gameGenieInputGrid.Add(gameGenieInput)
		gameGenieInputGrid.Add(gameGenieNameInput)
		gameGenieGrid.Add(gameGenieInputGrid)

		cm.genieList = container.NewVBox()
		gameGenieGrid.Add(cm.genieList)

		// create the add button
		gameGenieAddButton := widget.NewButton("Add", func() {
			if err := cm.genie.Load(gameGenieInput.Text, gameGenieNameInput.Text); err != nil {
				// TODO show error
			} else {
				// clear the input
				gameGenieInput.SetText("")
				gameGenieNameInput.SetText("")

				// refresh the list
				cm.refreshGenieCheats() // TODO only refresh the game genie list
			}
		})

		// add the input box to the grid

		gameGenieInputGrid.Add(gameGenieAddButton)

		cheatGrid.Add(gameGenieGrid)

		// create save button
		saveButton := widget.NewButton("Save", func() {
			if err := cm.genie.Save("genie.cheats"); err != nil {
				// TODO show error
				return
			}
		})

		// create load button
		loadButton := widget.NewButton("Load", func() {
			if err := cm.genie.LoadFile("genie.cheats"); err != nil {
				// TODO show error
				return
			}
			cm.refreshGenieCheats()
		})

		// create the button grid
		buttonGrid := container.NewGridWithColumns(2)
		buttonGrid.Add(saveButton)
		buttonGrid.Add(loadButton)

		cheatGrid.Add(buttonGrid)
	}
	// create a list for the game shark cheats if enabled
	if cm.shark != nil {
		// create a grid for the game shark
		gameSharkGrid := container.NewVBox()

		// create the title box with title and debug select
		gameSharkTitleBox := container.New(titleLayout{})
		gameSharkTitle := widget.NewLabel("Game Shark")
		gameSharkTitleBox.Add(gameSharkTitle)

		// create the debug checkbox
		debugCheckbox := widget.NewCheck("", func(checked bool) {
			cm.sharkSubheading = checked
			cm.refreshSharkCheats()
		})

		gameSharkTitleBox.Add(debugCheckbox)

		// add the title box to the shark grid
		gameSharkGrid.Add(gameSharkTitleBox)

		// create a grid for the input
		gameSharkInputGrid := container.NewGridWithRows(3)

		// create a textbox input for cheat code
		gameSharkInput := widget.NewEntry()
		gameSharkInput.SetPlaceHolder("Enter cheat code")
		// create a textbox input for cheat name
		gameSharkNameInput := widget.NewEntry()
		gameSharkNameInput.SetPlaceHolder("Enter cheat name")

		// add the input components to the grid
		gameSharkInputGrid.Add(gameSharkInput)
		gameSharkInputGrid.Add(gameSharkNameInput)

		// add the input grid to the shark grid
		gameSharkGrid.Add(gameSharkInputGrid)

		// create the cheat list using a label and checkbox
		/*sharkList := widget.NewList(
			func() int {
				return len(cm.shark.Codes)
			},
			func() fyne.CanvasObject {
				box := container.New(&cheatEntryLayout{})

				// create the text vbox
				textBox := container.NewVBox()

				// make the text box fill the space

				// add the name label
				nameLabel := widget.NewLabel("")
				nameLabel.TextStyle = fyne.TextStyle{Bold: true}
				textBox.Add(nameLabel)

				// add the subheader label
				subheaderLabel := widget.NewLabel("")
				subheaderLabel.TextStyle = fyne.TextStyle{Monospace: true}
				textBox.Add(subheaderLabel)

				// add the text box to the box
				box.Add(textBox)

				// add a spacer
				box.Add(canvas.NewRectangle(color.Transparent))

				box.Add(widget.NewCheck("", func(checked bool) {}))
				return box
			},
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				// get the box
				box := obj.(*fyne.Container)

				// get the text box
				textBox := box.Objects[0].(*fyne.Container)

				// get the name label
				nameLabel := textBox.Objects[0].(*widget.Label)

				// get the subheader label
				subheaderLabel := textBox.Objects[1].(*widget.Label)

				// get the checkbox
				checkBox := box.Objects[2].(*widget.Check)

				// set the name label
				nameLabel.SetText(cm.shark.Codes[id].Name)

				// set the subheader label
				subheaderLabel.SetText(fmt.Sprintf("Address 0x%04X -> %02x", cm.shark.Codes[id].Address, cm.shark.Codes[id].NewData))

				// set the checkbox
				checkBox.SetChecked(cm.shark.Codes[id].Enabled)
			},
		)*/

		// create the cheat list using a label and checkbox
		cm.sharkList = container.NewVBox()
		// add the list to the shark grid
		gameSharkGrid.Add(cm.sharkList)

		// create a button for adding the cheat
		gameSharkAddButton := widget.NewButton("Add", func() {
			// add the cheat to the list
			if err := cm.shark.Load(gameSharkInput.Text, gameSharkNameInput.Text); err != nil {
				// TODO: show error
			} else {
				// clear the input
				gameSharkInput.SetText("")
				gameSharkNameInput.SetText("")

				// refresh the list
				cm.refreshSharkCheats()
			}
		})

		// add the button to the grid
		gameSharkInputGrid.Add(gameSharkAddButton)

		cheatGrid.Add(gameSharkGrid)

		// create save button
		saveButton := widget.NewButton("Save", func() {
			if err := cm.shark.Save("shark.cheats"); err != nil {
				return // TODO: show error
			}
		})

		// create load button
		loadButton := widget.NewButton("Load", func() {
			if err := cm.shark.LoadFile("shark.cheats"); err != nil {
				fmt.Println(err)
				return // TODO: show error
			}

			cm.refreshSharkCheats()
		})

		// create a grid for the buttons
		buttonGrid := container.NewGridWithColumns(2)
		buttonGrid.Add(saveButton)
		buttonGrid.Add(loadButton)

		// add the button grid to the cheat grid
		cheatGrid.Add(buttonGrid)
	}

	go func() {
		for {
			<-events // cheat manager does not react to any events from the GameBoy
		}
	}()

	return nil
}

func (cm *CheatManager) refreshGenieCheats() {
	cm.genieList.RemoveAll()
	for _, code := range cm.genie.Codes {
		// copy the name so it can be used in the callback
		codeName := code.Name

		entryBox := container.New(&cheatEntryLayout{})

		// create the text vbox
		textBox := container.NewVBox()

		// add the name label
		nameLabel := widget.NewLabel(code.Name)
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		textBox.Add(nameLabel)

		// add the subheader label
		if cm.genieSubheading {
			subheaderLabel := widget.NewLabel(fmt.Sprintf("ROM 0x%04X: %02X -> %02X", code.Address, code.OldData, code.NewData))
			subheaderLabel.TextStyle = fyne.TextStyle{Monospace: true}
			textBox.Add(subheaderLabel)
		}

		// add the text box to the box
		entryBox.Add(textBox)

		// add checkbox
		entryBox.Add(widget.NewCheck("", func(checked bool) {
			if checked {
				cm.genie.Enable(codeName)
			} else {
				cm.genie.Disable(codeName)
			}
		}))

		// add the box to the list
		cm.genieList.Add(entryBox)
	}
}

func (cm *CheatManager) refreshSharkCheats() {
	cm.sharkList.RemoveAll()
	for _, code := range cm.shark.Codes {
		codeName := code.Name // copy the name so it can be used in the callback
		entryBox := container.New(&cheatEntryLayout{})

		// create the text vbox
		textBox := container.NewVBox()

		// add the name label
		nameLabel := widget.NewLabel(code.Name)
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		textBox.Add(nameLabel)

		// add the subheader label
		if cm.sharkSubheading {
			subheaderLabel := widget.NewLabel(fmt.Sprintf("RAM 0x%04X: %02X", code.Address, code.NewData))
			subheaderLabel.TextStyle = fyne.TextStyle{Monospace: true}
			textBox.Add(subheaderLabel)
		}

		// add the text box to the box
		entryBox.Add(textBox)

		entryBox.Add(widget.NewCheck("", func(b bool) {
			if b {
				err := cm.shark.Enable(codeName)
				if err != nil {
					panic(err)
					return
				}
			} else {
				err := cm.shark.Disable(codeName)
				if err != nil {
					panic(err)
					return
				}
			}
		}))

		cm.sharkList.Add(entryBox)
	}

}
