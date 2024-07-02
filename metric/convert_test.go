package metric

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
)

func compare[T any](t *testing.T, a, b T) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("invalid %v %v", a, b)
	}
}

func TestEscapeInterfaceName(t *testing.T) {
	compare(t, escapeInterfaceName("a/1.hello hello"), "a-1_hellohello")
}
func TestCalcurateDiff(t *testing.T) {
	compare(t, calcurateDiff(1, 2, 4), 1)
	compare(t, calcurateDiff(2, 2, 4), 0)
	compare(t, calcurateDiff(3, 2, 4), 3)
	compare(t, calcurateDiff(4, 2, 4), 2)
	compare(t, calcurateDiff(5, 2, 4), 1)
}

func dummyFn(rawMetrics []collector.MetricsDutum) []*mackerel.MetricValue {
	return nil
}

func Test_replaceSnapshot(t *testing.T) {
	replaceSnapshot([]collector.MetricsDutum{
		{
			IfIndex: 1,
			Mib:     "",
			IfName:  "eth0",
			Value:   1,
		},
	}, dummyFn)

	actual := prevSnapshot
	expected := []collector.MetricsDutum{
		{
			IfIndex: 1,
			Mib:     "",
			IfName:  "eth0",
			Value:   1,
		},
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("value is mismatch (-actual +expected):%s", diff)
	}
}

func Test_convert(t *testing.T) {
	prevSnapshot = []collector.MetricsDutum{
		{
			IfIndex: 1,
			Mib:     "ifHCInOctets",
			IfName:  "eth0",
			Value:   1,
		},
		{
			IfIndex: 1,
			Mib:     "ifHCOutOctets",
			IfName:  "eth0",
			Value:   math.MaxUint64,
		},
		{
			IfIndex: 1,
			Mib:     "ifInDiscards",
			IfName:  "eth0",
			Value:   0,
		},
	}

	actual := convert([]collector.MetricsDutum{
		{
			IfIndex: 1,
			Mib:     "ifHCInOctets",
			IfName:  "eth0",
			Value:   1,
		},
		{
			IfIndex: 1,
			Mib:     "ifHCOutOctets",
			IfName:  "eth0",
			Value:   60,
		},
		{
			IfIndex: 1,
			Mib:     "ifInDiscards",
			IfName:  "eth0",
			Value:   1,
		},
	})

	expected := []*mackerel.MetricValue{
		{
			Name:  "interface.eth0.rxBytes.delta",
			Time:  time.Now().Unix(),
			Value: uint64(0),
		},
		{
			Name:  "interface.eth0.txBytes.delta",
			Time:  time.Now().Unix(),
			Value: uint64(1),
		},
		{
			Name:  "custom.interface.ifInDiscards.eth0",
			Time:  time.Now().Unix(),
			Value: uint64(1),
		},
	}

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("value is mismatch (-actual +expected):%s", diff)
	}
}
