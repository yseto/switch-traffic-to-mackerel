package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/yseto/switch-traffic-to-mackerel/mib"
)

var loadedFilename string

type YAMLConfig struct {
	Community    string     `yaml:"community"`
	Target       string     `yaml:"target"`
	Interface    *Interface `yaml:"interface"`
	Mibs         []string   `yaml:"mibs"`
	SkipLinkdown bool       `yaml:"skip-linkdown"`
	Name         string     `yaml:"name"`
	Mackerel     *Mackerel  `yaml:"mackerel"`
	Debug        bool       `yaml:"debug"`
}

type Interface struct {
	Include *string `yaml:"include"`
	Exclude *string `yaml:"exclude"`
}

type Mackerel struct {
	HostID string `yaml:"host-id"`
	ApiKey string `yaml:"x-api-key"`
}

type Config struct {
	Community         string
	Target            string
	Name              string
	MIBs              []string
	IncludeRegexp     *regexp.Regexp
	ExcludeRegexp     *regexp.Regexp
	SkipDownLinkState bool
	Debug             bool
	Mackerel          *Mackerel
}

func (c *Config) HostIdPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, fmt.Sprintf("%s.id.txt", c.Target)), nil
}

func Init(filename string) (*Config, error) {
	loadedFilename = filename
	f, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var t YAMLConfig
	err = yaml.Unmarshal(f, &t)
	if err != nil {
		return nil, err
	}

	if t.Community == "" {
		return nil, fmt.Errorf("community is needed.")
	}
	if t.Target == "" {
		return nil, fmt.Errorf("target is needed.")
	}

	name := t.Name
	if name == "" {
		name = t.Target
	}

	c := &Config{
		Target:            t.Target,
		Community:         t.Community,
		SkipDownLinkState: t.SkipLinkdown,
		Name:              name,
		Debug:             t.Debug,
	}

	if t.Interface != nil {
		if t.Interface.Include != nil && t.Interface.Exclude != nil {
			return nil, fmt.Errorf("Interface.Exclude, Interface.Include is exclusive control.")
		}
		if t.Interface.Include != nil {
			c.IncludeRegexp, err = regexp.Compile(*t.Interface.Include)
			if err != nil {
				return nil, err
			}
		}
		if t.Interface.Exclude != nil {
			c.ExcludeRegexp, err = regexp.Compile(*t.Interface.Exclude)
			if err != nil {
				return nil, err
			}
		}
	}

	c.MIBs, err = mib.Validate(t.Mibs)
	if err != nil {
		return nil, err
	}

	if t.Mackerel != nil {
		c.Mackerel = t.Mackerel
	}

	return c, nil
}
