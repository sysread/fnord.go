package debug

import (
	"log"
	"os"
	"sync"
)

var (
	logFile  *os.File
	logMutex sync.Mutex
	logger   *log.Logger
)

func Init() error {
	var err error
	logFile, err = os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	logger = log.New(logFile, "", log.Ldate|log.Ltime)
	return nil
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

func Log(format string, v ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logger != nil {
		logger.Printf(format, v...)
	}
}
