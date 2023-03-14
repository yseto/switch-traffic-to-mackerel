package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"

	"github.com/maruel/natural"
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

func (m *MetricsDutum) String() string {
	return fmt.Sprintf("%d\t%s\t%s\t%d", m.IfIndex, m.IfName, m.Mib, m.Value)
}

type CollectParams struct {
	community, target, name      string
	mibs                         []string
	includeRegexp, excludeRegexp *regexp.Regexp
	skipDownLinkState            bool
}

var log = logrus.New()
var apikey = os.Getenv("MACKEREL_API_KEY")

func parseFlags() (*CollectParams, error) {
	var configFilename string
	flag.StringVar(&configFilename, "config", "config.yaml", "config `filename`")
	flag.Parse()

	f, err := os.ReadFile(configFilename)
	if err != nil {
		return nil, err
	}
	var t config.Config
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

	c := &CollectParams{
		target:            t.Target,
		community:         t.Community,
		skipDownLinkState: t.SkipLinkdown,
		name:              name,
	}

	if t.Interface != nil {
		if t.Interface.Include != nil && t.Interface.Exclude != nil {
			return nil, fmt.Errorf("Interface.Exclude, Interface.Include is exclusive control.")
		}
		if t.Interface.Include != nil {
			c.includeRegexp, err = regexp.Compile(*t.Interface.Include)
			if err != nil {
				return nil, err
			}
		}
		if t.Interface.Exclude != nil {
			c.excludeRegexp, err = regexp.Compile(*t.Interface.Exclude)
			if err != nil {
				return nil, err
			}
		}
	}

	c.mibs, err = mib.Validate(t.Mibs)
	if err != nil {
		return nil, err
	}

	return c, nil
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
		dutum, err := collect(ctx, collectParams)
		if err != nil {
			log.Fatal(err)
		}

		var dutumStr []string
		for i := range dutum {
			dutumStr = append(dutumStr, dutum[i].String())
		}
		sort.Sort(natural.StringSlice(dutumStr))
		fmt.Println(strings.Join(dutumStr, "\n"))
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
	if c.skipDownLinkState {
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
			if c.includeRegexp != nil && !c.includeRegexp.MatchString(ifName) {
				continue
			}

			if c.excludeRegexp != nil && c.excludeRegexp.MatchString(ifName) {
				continue
			}

			// skip when down(2)
			if c.skipDownLinkState && !ifOperStatus[ifIndex] {
				continue
			}

			metrics = append(metrics, MetricsDutum{IfIndex: ifIndex, Mib: mibName, IfName: ifName, Value: value})
		}
	}
	return metrics, nil
}
