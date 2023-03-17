package collector

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/maruel/natural"

	"github.com/yseto/switch-traffic-to-mackerel/config"
	"github.com/yseto/switch-traffic-to-mackerel/mib"
	"github.com/yseto/switch-traffic-to-mackerel/snmp"
)

type snmpClientImpl interface {
	BulkWalk(oid string, length uint64) (map[uint64]uint64, error)
	BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error)
	BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error)
	Close() error
	GetInterfaceNumber() (uint64, error)
}

func Do(ctx context.Context, c *config.Config) ([]MetricsDutum, error) {
	snmpClient, err := snmp.Init(ctx, c.Target, c.Community)
	if err != nil {
		return nil, err
	}
	defer snmpClient.Close()
	return do(ctx, snmpClient, c)
}

func do(ctx context.Context, snmpClient snmpClientImpl, c *config.Config) ([]MetricsDutum, error) {
	ifNumber, err := snmpClient.GetInterfaceNumber()
	if err != nil {
		return nil, err
	}
	ifDescr, err := snmpClient.BulkWalkGetInterfaceName(ifNumber)
	if err != nil {
		return nil, err
	}

	var ifOperStatus map[uint64]bool
	if c.SkipDownLinkState {
		ifOperStatus, err = snmpClient.BulkWalkGetInterfaceState(ifNumber)
		if err != nil {
			return nil, err
		}
	}

	metrics := make([]MetricsDutum, 0)

	for _, mibName := range c.MIBs {
		values, err := snmpClient.BulkWalk(mib.Oidmapping[mibName], ifNumber)
		if err != nil {
			return nil, err
		}

		for ifIndex, value := range values {
			ifName := ifDescr[ifIndex]
			if c.IncludeRegexp != nil && !c.IncludeRegexp.MatchString(ifName) {
				continue
			}

			if c.ExcludeRegexp != nil && c.ExcludeRegexp.MatchString(ifName) {
				continue
			}

			// skip when down(2)
			if c.SkipDownLinkState && !ifOperStatus[ifIndex] {
				continue
			}

			metrics = append(metrics, MetricsDutum{IfIndex: ifIndex, Mib: mibName, IfName: ifName, Value: value})
		}
	}
	if c.Debug {
		debugPrint(metrics)
	}
	return metrics, nil
}

func debugPrint(dutum []MetricsDutum) {
	var dutumStr []string
	for i := range dutum {
		dutumStr = append(dutumStr, dutum[i].String())
	}
	sort.Sort(natural.StringSlice(dutumStr))
	// debug print.
	fmt.Print("\033[H\033[2J")
	fmt.Println(time.Now().Format(time.ANSIC))
	fmt.Println(strings.Join(dutumStr, "\n"))
}
