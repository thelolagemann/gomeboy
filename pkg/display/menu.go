package display

type Menu struct {
	Options  []string
	Selected int
}

func NewMenu(options []string) *Menu {
	return &Menu{
		Options:  options,
		Selected: 0,
	}
}

func (m *Menu) SelectNext() {
	m.Selected = (m.Selected + 1) % len(m.Options)
}

func (m *Menu) SelectPrevious() {
	m.Selected = (m.Selected - 1 + len(m.Options)) % len(m.Options)
}

func (m *Menu) Select(index int) {
	m.Selected = index
}

func (m *Menu) SelectedOption() string {
	return m.Options[m.Selected]
}

func (m *Menu) Render() string {
	var s string
	for i, o := range m.Options {
		if i == m.Selected {
			s += "> "
		} else {
			s += "  "
		}
		s += o + ""
	}

	return s
}
