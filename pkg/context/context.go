package context

import (
	"github.com/sysread/fnord/pkg/config"
	"github.com/sysread/fnord/pkg/data"
	"github.com/sysread/fnord/pkg/gpt"
)

type Context struct {
	Config    *config.Config
	DataStore *data.DataStore
	GptClient *gpt.OpenAIClient
}

func NewContext() *Context {
	conf := config.Getopts()
	ds := data.NewDataStore(conf)
	gptClient := gpt.NewOpenAIClient(conf)

	return &Context{
		Config:    conf,
		DataStore: ds,
		GptClient: gptClient,
	}
}
