package collector

import (
	"context"

	"github.com/yseto/switch-traffic-to-mackerel/config"
	"github.com/yseto/switch-traffic-to-mackerel/mib"
	"github.com/yseto/switch-traffic-to-mackerel/snmp"
)

type snmpClientImpl interface {
	BulkWalk(oid string, length uint64) (map[uint64]uint64, error)
	BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error)
	BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error)
	BulkWalkGetInterfaceIPAddress() (map[uint64][]string, error)
	BulkWalkGetInterfacePhysAddress(length uint64) (map[uint64]string, error)
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
	return metrics, nil
}

func DoInterfaceIPAddress(ctx context.Context, c *config.Config) ([]Interface, error) {
	snmpClient, err := snmp.Init(ctx, c.Target, c.Community)
	if err != nil {
		return nil, err
	}
	defer snmpClient.Close()
	return doInterfaceIPAddress(ctx, snmpClient, c)
}

func doInterfaceIPAddress(ctx context.Context, snmpClient snmpClientImpl, c *config.Config) ([]Interface, error) {
	ifNumber, err := snmpClient.GetInterfaceNumber()
	if err != nil {
		return nil, err
	}
	ifDescr, err := snmpClient.BulkWalkGetInterfaceName(ifNumber)
	if err != nil {
		return nil, err
	}

	ifIndexIP, err := snmpClient.BulkWalkGetInterfaceIPAddress()
	if err != nil {
		return nil, err
	}

	ifPhysAddress, err := snmpClient.BulkWalkGetInterfacePhysAddress(ifNumber)
	if err != nil {
		return nil, err
	}

	var interfaces []Interface
	for ifIndex, ip := range ifIndexIP {
		if name, ok := ifDescr[ifIndex]; ok {
			phy := ifPhysAddress[ifIndex]
			interfaces = append(interfaces, Interface{
				IfName:     name,
				IpAddress:  ip,
				MacAddress: phy,
			})
		}
	}

	return interfaces, nil
}
