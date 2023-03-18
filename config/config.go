package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/yseto/switch-traffic-to-mackerel/mib"
)

var loadedFilename string

type YAMLConfig struct {
	Community    string     `yaml:"community"`
	Target       string     `yaml:"target"`
	Interface    *Interface `yaml:"interface,omitempty"`
	Mibs         []string   `yaml:"mibs,omitempty"`
	SkipLinkdown bool       `yaml:"skip-linkdown,omitempty"`
	Mackerel     *Mackerel  `yaml:"mackerel,omitempty"`
	Debug        bool       `yaml:"debug,omitempty"`
	DryRun       bool       `yaml:"dry-run,omitempty"`
}

type Interface struct {
	Include *string `yaml:"include,omitempty"`
	Exclude *string `yaml:"exclude,omitempty"`
}

type Mackerel struct {
	HostID            string `yaml:"host-id"`
	ApiKey            string `yaml:"x-api-key"`
	Name              string `yaml:"name,omitempty"`
	IgnoreNetworkInfo bool   `yaml:"ignore-network-info,omitempty"`
}

type Config struct {
	Community         string
	Target            string
	MIBs              []string
	IncludeRegexp     *regexp.Regexp
	ExcludeRegexp     *regexp.Regexp
	SkipDownLinkState bool
	Debug             bool
	DryRun            bool
	Mackerel          *Mackerel
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

	c := &Config{
		Target:            t.Target,
		Community:         t.Community,
		SkipDownLinkState: t.SkipLinkdown,
		Debug:             t.Debug,
		DryRun:            t.DryRun,
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
