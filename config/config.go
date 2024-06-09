package config

import (
	"cmp"
	"crypto/md5"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/mackerelio/mackerel-client-go"
	"github.com/yseto/switch-traffic-to-mackerel/mib"
)

var loadedFilename string

type YAMLConfig struct {
	Community    string       `yaml:"community"`
	Target       string       `yaml:"target"`
	Interface    *Interface   `yaml:"interface,omitempty"`
	Mibs         []string     `yaml:"mibs,omitempty"`
	SkipLinkdown bool         `yaml:"skip-linkdown,omitempty"`
	Mackerel     *Mackerel    `yaml:"mackerel,omitempty"`
	Debug        bool         `yaml:"debug,omitempty"`
	DryRun       bool         `yaml:"dry-run,omitempty"`
	CustomMibs   []*CustomMIB `yaml:"custom-mibs,omitempty"`
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

type CustomMIB struct {
	DisplayName string                `yaml:"display-name"`
	Unit        string                `yaml:"unit"`
	Mibs        []*MIBwithDisplayName `yaml:"mibs,omitempty"`
}

type MIBwithDisplayName struct {
	DisplayName string `yaml:"display-name"`
	MetricName  string `yaml:"metric-name,omitempty"`
	MIB         string `yaml:"mib"`
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

	CustomMIBs          []string
	CustomMIBsGraphDefs []*mackerel.GraphDefsParam
	// metricName:mib
	CustomMIBmetricNameMappedMIBs map[string]string
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
		return nil, fmt.Errorf("community is needed")
	}
	if t.Target == "" {
		return nil, fmt.Errorf("target is needed")
	}

	c := &Config{
		Target:                        t.Target,
		Community:                     t.Community,
		SkipDownLinkState:             t.SkipLinkdown,
		Debug:                         t.Debug,
		DryRun:                        t.DryRun,
		CustomMIBmetricNameMappedMIBs: map[string]string{},
	}

	if t.Interface != nil {
		if t.Interface.Include != nil && t.Interface.Exclude != nil {
			return nil, fmt.Errorf("Interface.Exclude, Interface.Include is exclusive control")
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

	for i := range t.CustomMibs {
		res, err := generateCustomMIB(t.CustomMibs[i])
		if err != nil {
			return nil, err
		}
		c.CustomMIBs = append(c.CustomMIBs, res.customMIBs...)
		c.CustomMIBsGraphDefs = append(c.CustomMIBsGraphDefs, res.graphDefs)
		for metricName, mib := range res.metricNameMappedMIBs {
			c.CustomMIBmetricNameMappedMIBs[metricName] = mib
		}
	}
	return c, nil
}

var metricRe = regexp.MustCompile("[-a-zA-Z0-9_]+")

func customMIBMackerelMetricNameParent(graphDisplayName string) string {
	a := md5.Sum([]byte(graphDisplayName))
	return fmt.Sprintf("custom.custommibs.%x", a)
}

func customMIBMackerelMetricName(graphDisplayName, metricName string) string {
	return fmt.Sprintf("%s.%s", customMIBMackerelMetricNameParent(graphDisplayName), metricName)
}

type customMIBConfig struct {
	customMIBs []string

	// metricName:MIB
	metricNameMappedMIBs map[string]string

	graphDefs *mackerel.GraphDefsParam
}

func generateCustomMIB(t *CustomMIB) (*customMIBConfig, error) {
	var customMIBs []string
	var metrics []*mackerel.GraphDefsMetric
	var metricNameMappedMIBs = make(map[string]string, 0)

	for idx := range t.Mibs {
		metricName := cmp.Or(t.Mibs[idx].MetricName, t.Mibs[idx].DisplayName)
		if !metricRe.MatchString(metricName) {
			return nil, fmt.Errorf("metricName is not valid : %s", metricName)
		}

		mackerelMetricName := customMIBMackerelMetricName(t.DisplayName, metricName)
		metrics = append(metrics, &mackerel.GraphDefsMetric{
			Name:        mackerelMetricName,
			DisplayName: t.Mibs[idx].DisplayName,
		})

		err := mib.ValidateCustom(t.Mibs[idx].MIB)
		if err != nil {
			return nil, err
		}
		customMIBs = append(customMIBs, t.Mibs[idx].MIB)

		metricNameMappedMIBs[mackerelMetricName] = t.Mibs[idx].MIB
	}

	return &customMIBConfig{
		graphDefs: &mackerel.GraphDefsParam{
			Name:        customMIBMackerelMetricNameParent(t.DisplayName),
			Unit:        t.Unit,
			DisplayName: t.DisplayName,
			Metrics:     metrics,
		},
		customMIBs:           customMIBs,
		metricNameMappedMIBs: metricNameMappedMIBs,
	}, nil
}
