package fyne

import "fyne.io/fyne/v2"

// MenuOption is used to customize the behaviour and properties of a [fyne.MenuItem]
type MenuOption func(*fyne.MenuItem)

// Checked allows toggling the state of a [fyne.MenuItem], calling onChange with the
// value whenever the [fyne.MenuItem] is clicked/tapped.
func Checked(b bool, onChange func()) MenuOption {
	return func(item *fyne.MenuItem) {
		tempFn := item.Action
		item.Action = func() {
			tempFn()
			item.Checked = !item.Checked
			onChange()
		}
		item.Checked = b
	}
}

// Gated disables the [fyne.MenuItem] from being interactable when b is false. The menu
// should be recreated whenever these conditions change (TODO make it so u dont)
func Gated(b bool) MenuOption {
	return func(item *fyne.MenuItem) {
		item.Disabled = b
	}
}

// NewCustomizedMenuItem creates a [fyne.MenuItem] with the provided label and fn, and applies
// all of the MenuOption(s) to it.
func NewCustomizedMenuItem(label string, fn func(), opts ...MenuOption) *fyne.MenuItem {
	m := fyne.NewMenuItem(label, fn)
	for _, o := range opts {
		o(m)
	}
	return m
}
