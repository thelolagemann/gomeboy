package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"github.com/thelolagemann/gomeboy/pkg/utils"
)

type labelEntryWithButton struct {
}

func (l *labelEntryWithButton) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 3 {
		return
	}
	objects[0].Resize(fyne.NewSize(100, 30))
	objects[1].Resize(fyne.NewSize(size.Width-100-25, 30))
	objects[2].Resize(fyne.NewSize(30, 30))

	objects[0].Move(fyne.NewPos(0, 0))
	objects[1].Move(fyne.NewPos(110, 0))
	objects[2].Move(fyne.NewPos(size.Width-25, 0))
}

func (l *labelEntryWithButton) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(650, 35)
}

type Settings struct {
	fyne.Preferences
	isAsking bool
}

func (s *Settings) Title() string {
	return "Settings"
}

func (s *Settings) Run(window fyne.Window, events <-chan event.Event) error {
	// create the settings view
	settingsView := container.NewVBox()

	// create the settings window
	window.SetContent(settingsView)

	// basic settings (GameBoy boot ROM, GameBoy Color boot ROM, GameBoy Model, etc.)
	basicSettings := container.NewVBox()

	// boot rom (string with a file picker)
	bootROM := container.New(&labelEntryWithButton{})
	bootROMLabel := widget.NewLabel("DMG Boot ROM")
	bootROMLabel.TextStyle.Monospace = true
	bootROMPath := widget.NewEntry()
	bootROMPath.SetText(s.StringWithFallback("bootroms.dmg", ""))

	bootROMButton := widget.NewButton("...", func() {
		// dirty hack to prevent the user from spamming the button
		if s.isAsking {
			return
		}

		s.isAsking = true
		defer func() {
			s.isAsking = false
		}()

		bootROMLocation, err := utils.AskForFile("Choose a DMG Boot ROM", "bootroms")
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		// validate the boot rom (just check the size for now)
		if !utils.IsSize(bootROMLocation, 256) {
			dialog.ShowError(fmt.Errorf("invalid boot rom size"), window)
			return
		}

		// update the boot rom path and save it to the preferences
		bootROMPath.SetText(bootROMLocation)
		s.SetString("bootroms.dmg", bootROMLocation)
	})

	bootROM.Add(bootROMLabel)
	bootROM.Add(bootROMPath)
	bootROM.Add(bootROMButton)

	basicSettings.Add(bootROM)

	// gameboy color boot rom (string with a file picker)
	gbcBootROM := container.New(&labelEntryWithButton{})
	gbcBootROMLabel := widget.NewLabel("CGB Boot ROM")
	gbcBootROMLabel.TextStyle.Monospace = true
	gbcBootROMPath := widget.NewEntry()
	gbcBootROMPath.SetText(s.StringWithFallback("bootroms.cgb", ""))

	gbcBootROMButton := widget.NewButton("...", func() {
		if s.isAsking {
			return
		}
		s.isAsking = true
		defer func() {
			s.isAsking = false
		}()
		gbcBootROMLocation, err := utils.AskForFile("Choose a CGB Boot ROM", "bootroms")
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		// validate the boot rom (just check the size for now)
		if !utils.IsSize(gbcBootROMLocation, 2304) {
			dialog.ShowError(fmt.Errorf("invalid boot rom size"), window)
			return
		}

		// update the boot rom path and save it to the preferences
		gbcBootROMPath.SetText(gbcBootROMLocation)
		s.SetString("bootroms.cgb", gbcBootROMLocation)
	})

	gbcBootROM.Add(gbcBootROMLabel)
	gbcBootROM.Add(gbcBootROMPath)
	gbcBootROM.Add(gbcBootROMButton)

	basicSettings.Add(gbcBootROM)

	// gameboy model (dropdown)
	gameboyModel := container.NewHBox()
	gameboyModelLabel := widget.NewLabel("Model")
	gameboyModelDropdown := widget.NewSelect([]string{"GameBoy", "GameBoy Color", "GameBoy Pocket"}, func(string) {

	})

	gameboyModel.Add(gameboyModelLabel)
	gameboyModel.Add(gameboyModelDropdown)

	basicSettings.Add(gameboyModel)

	// add the basic settings to the settings view
	settingsView.Add(basicSettings)

	runUntilQuit(events, func() {

	})
	return nil
}
