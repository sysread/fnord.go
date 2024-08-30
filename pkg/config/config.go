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
	OpenAIAsstId string
}

func Getopts() *Config {
	config := &Config{}

	config.
		SetOpenAIApiKey().
		SetOpenAIAsstId().
		SetHomeFromEnv().
		ReadCommandLineOptions()

	return config.
		validateOpenAIApiKey().
		validateOpenAIAsstId().
		validateBoxPath()
}

//------------------------------------------------------------------------------
// Setters
//------------------------------------------------------------------------------

func (c *Config) SetOpenAIApiKey() *Config {
	c.OpenAIApiKey = os.Getenv("FNORD_OPENAI_API_KEY")
	return c
}

func (c *Config) SetOpenAIAsstId() *Config {
	c.OpenAIAsstId = os.Getenv("FNORD_OPENAI_ASST_ID")
	return c
}

func (c *Config) SetHomeFromEnv() *Config {
	// Determine the base directory.
	basePath := os.Getenv("FNORD_HOME")
	if basePath == "" {
		basePath = filepath.Join(os.Getenv("HOME"), ".config", "fnord")
	}

	// Ensure the base directory exists.
	if !makeDir(basePath) {
		die("Could not create base directory (%s)", basePath)
	}

	c.Home = basePath
	return c
}

func (c *Config) ReadCommandLineOptions() *Config {
	pflag.StringVar(&c.Box, "box", DefaultBox, "boxes are isolated workspaces; conversations held within a box are isolated from other boxes")
	pflag.Parse()
	return c
}

//------------------------------------------------------------------------------
// Validation
//------------------------------------------------------------------------------

func (c *Config) validateOpenAIApiKey() *Config {
	if c.OpenAIApiKey == "" {
		die("FNORD_OPENAI_API_KEY must be set in the shell environment")
	}

	return c
}

func (c *Config) validateOpenAIAsstId() *Config {
	if c.OpenAIAsstId == "" {
		die("FNORD_OPENAI_ASST_ID must be set in the shell environment")
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

//------------------------------------------------------------------------------
// Helper functions
//------------------------------------------------------------------------------

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
