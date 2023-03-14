package collector

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/maruel/natural"

	"github.com/yseto/switch-traffic-to-mackerel/config"
	"github.com/yseto/switch-traffic-to-mackerel/mib"
	"github.com/yseto/switch-traffic-to-mackerel/snmp"
)

func Do(ctx context.Context, c *config.Collector) ([]MetricsDutum, error) {
	snmpClient, err := snmp.Init(ctx, c.Target, c.Community)
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
	return metrics, nil
}

func debugPrint(dutum []MetricsDutum) {
	var dutumStr []string
	for i := range dutum {
		dutumStr = append(dutumStr, dutum[i].String())
	}
	sort.Sort(natural.StringSlice(dutumStr))
	fmt.Println(strings.Join(dutumStr, "\n"))
}
