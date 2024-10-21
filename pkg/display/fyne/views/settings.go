package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Settings struct {
	widget.BaseWidget
	categories []string
	selected   int
	content    map[string]fyne.CanvasObject
}

func NewSettings(categories []string, content map[string]fyne.CanvasObject) *Settings {
	s := &Settings{categories: categories, content: content}
	s.ExtendBaseWidget(s)
	return s
}

func (s *Settings) CreateRenderer() fyne.WidgetRenderer {
	list := widget.NewList(
		func() int { return len(s.categories) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		}, func(id widget.ListItemID, object fyne.CanvasObject) {
			object.(*widget.Label).SetText(s.categories[id])
		})

	list.OnSelected = func(id widget.ListItemID) {
		s.selected = id
		s.Refresh()
	}

	contentContainer := s.content[s.categories[s.selected]]

	split := container.NewHSplit(list, contentContainer)
	split.Offset = 0.3
	return &settingsViewRenderer{
		view:    s,
		list:    list,
		content: contentContainer,
		split:   split,
	}
}

// settingsViewRenderer handles the actual layout and update of the SettingsView
type settingsViewRenderer struct {
	view    *Settings
	list    *widget.List
	content fyne.CanvasObject
	split   *container.Split
}

func (r *settingsViewRenderer) MinSize() fyne.Size {
	return r.split.MinSize()
}

func (r *settingsViewRenderer) Layout(size fyne.Size) {
	r.split.Resize(size)
}

func (r *settingsViewRenderer) Refresh() {
	r.content = r.view.content[r.view.categories[r.view.selected]] // Update the content to reflect the selected category
	r.split.Trailing = r.content
	r.split.Refresh()
}

func (r *settingsViewRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.split}
}

func (r *settingsViewRenderer) Destroy() {}
