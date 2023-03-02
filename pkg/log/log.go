package log

import "fmt"

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type logger struct {
}

func New() Logger {
	return &logger{}
}

func (l *logger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO]\t"+format+"\n", args...)
}

func (l *logger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR]\t"+format+"\n", args...)
}

func (l *logger) Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG]\t"+format+"\n", args...)
}
