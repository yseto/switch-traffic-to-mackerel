package collector

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/yseto/switch-traffic-to-mackerel/config"
)

type mockSnmpClient struct {
}

var errInvalid = errors.New("invalid error")

func (m *mockSnmpClient) BulkWalk(oid string, length uint64) (map[uint64]uint64, error) {
	switch oid {
	case "1.3.6.1.2.1.31.1.1.1.6":
		return map[uint64]uint64{
			1: 60,
			2: 60,
			3: 60,
			4: 60,
		}, nil
	case "1.3.6.1.2.1.31.1.1.1.10":
		return map[uint64]uint64{
			1: 120,
			2: 120,
			3: 120,
			4: 120,
		}, nil
	default:
		return nil, errInvalid
	}
}
func (m *mockSnmpClient) BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error) {
	return map[uint64]string{
		1: "lo0",
		2: "eth0",
		3: "eth1",
		4: "eth2",
	}, nil
}
func (m *mockSnmpClient) BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error) {
	return map[uint64]bool{
		1: true,
		2: false,
		3: true,
		4: true,
	}, nil
}
func (m *mockSnmpClient) Close() error {
	return nil
}
func (m *mockSnmpClient) GetInterfaceNumber() (uint64, error) {
	return 4, nil
}

func (m *mockSnmpClient) BulkWalkGetInterfaceIPAddress() (map[uint64][]string, error) {
	return map[uint64][]string{
		1: {"127.0.0.1"},
		2: {"192.0.2.1"},
		3: {"192.0.2.2", "192.0.2.3"},
		4: {"198.51.100.1"},
		5: {"198.51.100.2"},
	}, nil
}

func TestDo(t *testing.T) {
	ctx := context.Background()

	t.Run("non skip", func(t *testing.T) {
		c := &config.Config{
			MIBs: []string{"ifHCInOctets", "ifHCOutOctets"},
		}
		actual, err := do(ctx, &mockSnmpClient{}, c)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 1, Mib: "ifHCInOctets", IfName: "lo0", Value: 60},
			{IfIndex: 2, Mib: "ifHCInOctets", IfName: "eth0", Value: 60},
			{IfIndex: 3, Mib: "ifHCInOctets", IfName: "eth1", Value: 60},
			{IfIndex: 4, Mib: "ifHCInOctets", IfName: "eth2", Value: 60},
			{IfIndex: 1, Mib: "ifHCOutOctets", IfName: "lo0", Value: 120},
			{IfIndex: 2, Mib: "ifHCOutOctets", IfName: "eth0", Value: 120},
			{IfIndex: 3, Mib: "ifHCOutOctets", IfName: "eth1", Value: 120},
			{IfIndex: 4, Mib: "ifHCOutOctets", IfName: "eth2", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})

	t.Run("skip include", func(t *testing.T) {
		c := &config.Config{
			MIBs:          []string{"ifHCInOctets", "ifHCOutOctets"},
			IncludeRegexp: regexp.MustCompile("lo?"),
		}
		actual, err := do(ctx, &mockSnmpClient{}, c)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 1, Mib: "ifHCInOctets", IfName: "lo0", Value: 60},
			{IfIndex: 1, Mib: "ifHCOutOctets", IfName: "lo0", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})
	t.Run("skip exclude", func(t *testing.T) {
		c := &config.Config{
			MIBs:          []string{"ifHCInOctets", "ifHCOutOctets"},
			ExcludeRegexp: regexp.MustCompile("0$"),
		}
		actual, err := do(ctx, &mockSnmpClient{}, c)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 3, Mib: "ifHCInOctets", IfName: "eth1", Value: 60},
			{IfIndex: 4, Mib: "ifHCInOctets", IfName: "eth2", Value: 60},
			{IfIndex: 3, Mib: "ifHCOutOctets", IfName: "eth1", Value: 120},
			{IfIndex: 4, Mib: "ifHCOutOctets", IfName: "eth2", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})

	t.Run("skip down-linkstate", func(t *testing.T) {
		c := &config.Config{
			MIBs:              []string{"ifHCInOctets", "ifHCOutOctets"},
			SkipDownLinkState: true,
		}
		actual, err := do(ctx, &mockSnmpClient{}, c)
		if err != nil {
			t.Error("invalid raised error")
		}
		expected := []MetricsDutum{
			{IfIndex: 1, Mib: "ifHCInOctets", IfName: "lo0", Value: 60},
			{IfIndex: 3, Mib: "ifHCInOctets", IfName: "eth1", Value: 60},
			{IfIndex: 4, Mib: "ifHCInOctets", IfName: "eth2", Value: 60},
			{IfIndex: 1, Mib: "ifHCOutOctets", IfName: "lo0", Value: 120},
			{IfIndex: 3, Mib: "ifHCOutOctets", IfName: "eth1", Value: 120},
			{IfIndex: 4, Mib: "ifHCOutOctets", IfName: "eth2", Value: 120},
		}
		if d := cmp.Diff(
			actual,
			expected,
			cmpopts.SortSlices(func(i, j MetricsDutum) bool { return i.String() < j.String() }),
		); d != "" {
			t.Errorf("invalid result %s", d)
		}
	})

}
