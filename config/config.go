package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type Config struct {
	Community    string     `yaml:"community"`
	Target       string     `yaml:"target"`
	Interface    *Interface `yaml:"interface"`
	Mibs         []string   `yaml:"mibs"`
	SkipLinkdown bool       `yaml:"skip-linkdown"`
	Name         string     `yaml:"name"`
	Mackerel     *Mackerel  `yaml:"mackerel"`
}

type Interface struct {
	Include *string `yaml:"include"`
	Exclude *string `yaml:"exclude"`
}

type Mackerel struct {
	HostID string `yaml:"host-id"`
	ApiKey string `yaml:"x-api-key"`
}

type Collector struct {
	Community         string
	Target            string
	Name              string
	MIBs              []string
	IncludeRegexp     *regexp.Regexp
	ExcludeRegexp     *regexp.Regexp
	SkipDownLinkState bool
}

func (c *Collector) HostIdPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, fmt.Sprintf("%s.id.txt", c.Target)), nil
}
