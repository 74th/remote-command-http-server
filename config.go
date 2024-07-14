package remotecommandhttpserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MaxConcurrency int         `json:"max_concurrency,omitempty" yaml:"max_concurrency,omitempty"`
	Cmds           []CmdConfig `json:"cmds" yaml:"cmds"`
}

type PathArgs struct {
	Name        string
	PlaceHolder string
}

type CmdConfig struct {
	Path     string            `json:"path" yaml:"path"`
	Cmd      []string          `json:"cmd" yaml:"cmd"`
	Cwd      string            `json:"cwd,omitempty" yaml:"cwd,omitempty"`
	Envs     map[string]string `json:"envs,omitempty" yaml:"envs,omitempty"`
	EnvFile  string            `json:"env_file,omitempty" yaml:"env_file,omitempty"`
	PathArgs []PathArgs
}

func LoadConfig(path string) (*Config, error) {
	ext := filepath.Ext(path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	conf := &Config{}

	if ext == ".json" {
		if err := json.NewDecoder(f).Decode(conf); err != nil {
			return nil, fmt.Errorf("failed to decode json: %w", err)
		}
	} else if ext == ".yaml" || ext == ".yml" {
		if err := yaml.NewDecoder(f).Decode(conf); err != nil {
			return nil, fmt.Errorf("failed to decode yaml: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unrecognized extension: %s", ext)
	}

	conf.parsePathArgs()

	return conf, nil
}

func (c *Config) parsePathArgs() {
	re, err := regexp.Compile(`\{([^}]+)\}`)
	if err != nil {
		panic(err.Error())
	}

	for i := range c.Cmds {
		cmd := &c.Cmds[i]

		matches := re.FindAllStringSubmatch(cmd.Path, -1)
		for _, match := range matches {
			cmd.PathArgs = append(
				cmd.PathArgs,
				PathArgs{
					Name:        match[1],
					PlaceHolder: match[0],
				},
			)
		}
	}
}

func (c *Config) Validate() error {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	for i, cmd := range c.Cmds {
		if !strings.HasPrefix(cmd.Path, "/") {
			return fmt.Errorf("cmds[%d].path must be start with /: %s", i, cmd.Path)
		}
		if len(cmd.Cmd) == 0 {
			return fmt.Errorf("cmds[%d].cmd must not be empty", i)
		}
		for j, arg := range cmd.Cmd {
			if len(arg) == 0 {
				return fmt.Errorf("cmds[%d].cmd[%d] must not be empty", i, j)
			}
		}

		if cmd.Cwd == "" {
			cmd.Cwd = cwd
		}
	}

	return nil
}
