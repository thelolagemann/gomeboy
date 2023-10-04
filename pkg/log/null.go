package log

// nullLogger is a logger that does nothing.
type nullLogger struct{}

func (n nullLogger) Fatal(str string) {
}

func (n nullLogger) Infof(format string, args ...interface{}) {
}

func (n nullLogger) Errorf(format string, args ...interface{}) {
}

func (n nullLogger) Debugf(format string, args ...interface{}) {
}

// NewNullLogger returns a logger that does nothing.
func NewNullLogger() Logger {
	return &nullLogger{}
}
