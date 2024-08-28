package main

import (
	"github.com/sysread/fnord/pkg/debug"
	"github.com/sysread/fnord/pkg/ui"
)

func main() {
	// Initialize the debug logger
	if err := debug.Init(); err != nil {
		panic(err)
	}

	defer debug.Close()
	debug.Log("\n\n\n\n\n\n\n\n\n\n")

	app := ui.New()
	app.Run()
}
