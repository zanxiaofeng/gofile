package log

import "fmt"

const (
	LevelVerbose = iota
	LevelNormal
	LevelError
)

var Level = LevelNormal

func Debug(msg ...interface{}) {
	if Level == LevelVerbose {
		fmt.Println(msg...)
	}
}

func Normal(msg ...interface{}) {
	if Level < LevelError {
		fmt.Println(msg...)
	}
}

func Err(msg ...interface{}) {
	fmt.Println(msg...)
}
