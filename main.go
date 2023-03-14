package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/yseto/switch-traffic-to-mackerel/config"
	"github.com/yseto/switch-traffic-to-mackerel/mib"
	"github.com/yseto/switch-traffic-to-mackerel/snmp"
)

type MetricsDutum struct {
	IfIndex uint64 `json:"ifIndex"`
	Mib     string `json:"mib"`
	IfName  string `json:"ifName"`
	Value   uint64 `json:"value"`
}

type CollectParams struct {
	community, target, name            string
	mibs                               []string
	includeRegexp, excludeRegexp       *regexp.Regexp
	includeInterface, excludeInterface *string
	skipDownLinkState                  *bool
}

var log = logrus.New()
var apikey = os.Getenv("MACKEREL_API_KEY")

func parseFlags() (*CollectParams, error) {
	var community, target, name string
	var includeInterface, excludeInterface *string
	level := flag.Bool("verbose", false, "verbose")
	skipDownLinkState := flag.Bool("skip-down-link-state", false, "skip down link state")
	flag.StringVar(&name, "name", "", "name")
	var configFilename string
	flag.StringVar(&configFilename, "config", "config.yaml", "config `filename`")
	flag.Parse()

	f, err := os.ReadFile(configFilename)
	if err != nil {
		log.Fatal(err)
	}
	var t config.Config
	err = yaml.Unmarshal(f, &t)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if t.Community == "" {
		log.Fatal("community is needed.")
	}
	community = t.Community
	if t.Target == "" {
		log.Fatal("target is needed.")
	}
	target = t.Target

	if t.Interface != nil {
		includeInterface = t.Interface.Include
		excludeInterface = t.Interface.Exclude
	}

	logLevel := logrus.WarnLevel
	if *level {
		logLevel = logrus.DebugLevel
	}
	log.SetLevel(logLevel)

	if name == "" {
		name = target
	}

	if includeInterface != nil && excludeInterface != nil {
		return nil, errors.New("excludeInterface, includeInterface is exclusive control.")
	}
	var includeReg *regexp.Regexp
	if includeInterface != nil {
		includeReg, err = regexp.Compile(*includeInterface)
		if err != nil {
			return nil, err
		}
	}
	var excludeReg *regexp.Regexp
	if excludeInterface != nil {
		excludeReg, err = regexp.Compile(*excludeInterface)
		if err != nil {
			return nil, err
		}
	}

	mibs, err := mib.Validate(t.Mibs)
	if err != nil {
		return nil, err
	}

	return &CollectParams{
		target:            target,
		name:              name,
		community:         community,
		mibs:              mibs,
		includeRegexp:     includeReg,
		excludeRegexp:     excludeReg,
		includeInterface:  includeInterface,
		excludeInterface:  excludeInterface,
		skipDownLinkState: skipDownLinkState,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	collectParams, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	log.Info("start")

	if apikey == "" {
		log.SetLevel(logrus.DebugLevel)

		_, err := collect(ctx, collectParams)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		runMackerel(ctx, collectParams)
	}
}

func collect(ctx context.Context, c *CollectParams) ([]MetricsDutum, error) {
	snmpClient, err := snmp.Init(ctx, c.target, c.community)
	if err != nil {
		return nil, err
	}
	defer snmpClient.Close()

	ifNumber, err := snmpClient.GetInterfaceNumber()
	if err != nil {
		return nil, err
	}
	ifDescr, err := snmpClient.BulkWalkGetInterfaceName(ifNumber)
	if err != nil {
		return nil, err
	}

	var ifOperStatus map[uint64]bool
	if *c.skipDownLinkState {
		ifOperStatus, err = snmpClient.BulkWalkGetInterfaceState(ifNumber)
		if err != nil {
			return nil, err
		}
	}

	metrics := make([]MetricsDutum, 0)

	for _, mibName := range c.mibs {
		values, err := snmpClient.BulkWalk(mib.Oidmapping[mibName], ifNumber)
		if err != nil {
			return nil, err
		}

		for ifIndex, value := range values {
			ifName := ifDescr[ifIndex]
			if c.includeInterface != nil && !c.includeRegexp.MatchString(ifName) {
				continue
			}

			if c.excludeInterface != nil && c.excludeRegexp.MatchString(ifName) {
				continue
			}

			// skip when down(2)
			if *c.skipDownLinkState && !ifOperStatus[ifIndex] {
				continue
			}

			log.WithFields(logrus.Fields{
				"IfIndex": ifIndex,
				"Mib":     mibName,
				"IfName":  ifName,
				"Value":   value,
			}).Debug()

			metrics = append(metrics, MetricsDutum{IfIndex: ifIndex, Mib: mibName, IfName: ifName, Value: value})
		}
	}
	return metrics, nil
}
