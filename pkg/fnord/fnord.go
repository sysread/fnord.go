package fnord

import (
	"os"

	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/console"
	"github.com/sysread/fnord/pkg/gpt"
	"github.com/sysread/fnord/pkg/storage"
)

type Fnord struct {
	Config    *config.Config
	GptClient *gpt.OpenAIClient
}

func NewFnord() *Fnord {
	conf := config.Getopts()
	gptClient := gpt.NewOpenAIClient(conf)

	err := storage.Init(conf)
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list-boxes":
			console.ListBoxes()
			os.Exit(0)
		case "list-projects":
			console.ListProjects()
			os.Exit(0)
		}
	}

	return &Fnord{
		Config:    conf,
		GptClient: gptClient,
	}
}
