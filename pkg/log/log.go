package log

import (
	"fmt"
	"os"
)

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Fatal(str string)
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

func (l *logger) Fatal(str string) {
	fmt.Printf("[FATAL]\t" + str + "\n")
	os.Exit(1)
}

var l = New()

func Infof(format string, args ...interface{}) {
	l.Infof(format, args...)
}

func Errorf(format string, args ...interface{}) {
	l.Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	l.Debugf(format, args...)
}

func Fatal(str string) {
	l.Fatal(str)
}
