// Package log provides a simple interface for printing messages to stdout.
package log

import "fmt"

const (
	levelQuiet = iota
	levelNormal
	levelVerbose
)

var level = levelNormal

// SetQuiet sets the log level to quite. Only Error prints. All other logs are
// ignored.
func SetQuiet() {
	level = levelQuiet
}

// SetNormal sets the log level to normal. Info logs are ignored.
func SetNormal() {
	level = levelNormal
}

// SetVerbose sets the log level to verbose. All messages are printed.
func SetVerbose() {
	level = levelVerbose
}

// Info formats using the default formats for its operands and writes to
// standard output if level is levelVerbose.
func Info(a ...interface{}) {
	if level == levelVerbose {
		fmt.Println(a...)
	}
}

// Normal formats using the default formats for its operands and writes to
// standard output if level greater than or equal to levelNormal.
func Normal(a ...interface{}) {
	if level >= levelNormal {
		fmt.Println(a...)
	}
}

// Error formats using the default formats for its operands and writes to
// standard output, ignoring level.
func Error(a ...interface{}) {
	fmt.Println(a...)
}
