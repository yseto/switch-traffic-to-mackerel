package mackerel

import (
	"container/list"
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

func TestEnqueue(t *testing.T) {
	t.Run("replace Snapshot", func(t *testing.T) {
		queue := &Queue{
			buffers: list.New(),
			Snapshot: []collector.MetricsDutum{
				{
					IfIndex: 1,
					Mib:     "",
					IfName:  "eth0",
					Value:   1,
				},
			},
		}
		newSnapshot := []collector.MetricsDutum{
			{
				IfIndex: 1,
				Mib:     "",
				IfName:  "eth0",
				Value:   1,
			},
		}
		queue.Enqueue(newSnapshot)

		if !reflect.DeepEqual(queue.Snapshot, newSnapshot) {
			t.Error("replace Snapshot is invalid")
		}
	})

	t.Run("calcurate", func(t *testing.T) {
		queue := &Queue{
			buffers: list.New(),
			Snapshot: []collector.MetricsDutum{
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
			},
		}
		newSnapshot := []collector.MetricsDutum{
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
		}
		queue.Enqueue(newSnapshot)

		e := queue.buffers.Front()

		actual := e.Value.([](*mackerel.MetricValue))
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
			t.Errorf("failed transform %s", diff)
		}
	})

}
