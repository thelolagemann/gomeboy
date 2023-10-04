package views

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/thelolagemann/gomeboy/pkg/display/event"
	"sync"
)

type Log struct {
	sync.RWMutex

	entries []string
}

func (l *Log) Title() string {
	return "Log"
}

func (l *Log) Infof(format string, args ...interface{}) {
	l.Lock()
	defer l.Unlock()

	l.entries = append(l.entries, "[INFO]"+fmt.Sprintf(format, args...))
}

func (l *Log) Errorf(format string, args ...interface{}) {
	l.Lock()
	defer l.Unlock()

	l.entries = append(l.entries, "[ERROR]"+fmt.Sprintf(format, args...))
}

func (l *Log) Debugf(format string, args ...interface{}) {
	l.Lock()
	defer l.Unlock()

	l.entries = append(l.entries, "[DEBUG]"+fmt.Sprintf(format, args...))
}

func (l *Log) Run(window fyne.Window, events <-chan event.Event) error {
	// create a log view
	view := container.NewVBox()

	// set the content of the window
	window.SetContent(view)

	// handle event
	go func() {
		for {
			select {
			case <-events:
				if len(l.entries) == 0 {
					continue
				}
				l.Lock()
				for _, entry := range l.entries {
					view.Add(container.NewHBox(widget.NewLabel(entry)))
				}
				// clear the entries
				l.entries = []string{}

				// make sure there's not too many entries
				if len(view.Objects) > 20 {
					view.Objects = view.Objects[len(view.Objects)-20:]
				}
				l.Unlock()

				// refresh the window
				view.Refresh()
			}
		}
	}()

	return nil
}
