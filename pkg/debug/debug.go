package debug

import (
	"fmt"
)

var LogChannel = make(chan string, 100)

func Log(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	LogChannel <- msg
}
