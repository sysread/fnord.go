package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
)

const (
	DefaultBox = "default"
)

type Config struct {
	Home         string
	Box          string
	BoxPath      string
	OpenAIApiKey string
}

func Getopts() *Config {
	config := &Config{
		OpenAIApiKey: os.Getenv("OPENAI_API_KEY"),
	}

	basePath := os.Getenv("FNORD_HOME")
	if basePath == "" {
		basePath = filepath.Join(os.Getenv("HOME"), ".config", "fnord")
	}

	// Ensure the base directory exists.
	if !makeDir(basePath) {
		die("Could not create base directory (%s)", basePath)
	}

	config.Home = basePath

	pflag.StringVar(&config.Box, "box", DefaultBox, "boxes are isolated workspaces; conversations held within a box are isolated from other boxes")
	pflag.Parse()

	return config.
		validateOpenAIApiKey().
		validateBoxPath()
}

func (c *Config) validateOpenAIApiKey() *Config {
	if c.OpenAIApiKey == "" {
		die("OPENAI_API_KEY must be set in the shell environment")
	}

	return c
}

func (c *Config) validateBoxPath() *Config {
	if c.Box == "" {
		die("Box name cannot be empty")
	}

	if strings.Contains(c.Box, "/") {
		die("Box name cannot contain '/'")
	}

	c.BoxPath = path.Join(c.Home, c.Box)

	if !makeDir(c.BoxPath) {
		die("Could not create box directory (%s)", c.BoxPath)
	}

	return c
}

func die(fmtString string, args ...interface{}) {
	panic(fmt.Sprintf(fmtString, args...))
}

func pathExists(path string) bool {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}

func makeDir(path string) bool {
	if err := os.MkdirAll(path, 0700); err != nil {
		die("Could not create directory (%s): %s", path, err)
	}

	return pathExists(path)
}
