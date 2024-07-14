package metric

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"github.com/yseto/switch-traffic-to-mackerel/collector"
)

var prevSnapshot []collector.MetricsDutum

func Convert(rawMetrics []collector.MetricsDutum) []*mackerel.MetricValue {
	return replaceSnapshot(rawMetrics, convert)
}

func replaceSnapshot(rawMetrics []collector.MetricsDutum, fn func(rawMetrics []collector.MetricsDutum) []*mackerel.MetricValue) []*mackerel.MetricValue {
	defer func() {
		prevSnapshot = rawMetrics
	}()

	if len(prevSnapshot) == 0 {
		return nil
	}
	return fn(rawMetrics)
}

func convert(rawMetrics []collector.MetricsDutum) []*mackerel.MetricValue {
	now := time.Now().Unix()

	metrics := make([]*mackerel.MetricValue, 0)
	for _, metric := range rawMetrics {
		prevValue := metric.Value
		for _, v := range prevSnapshot {
			if v.IfIndex == metric.IfIndex && v.Mib == metric.Mib {
				prevValue = v.Value
				break
			}
		}

		value := calcurateDiff(prevValue, metric.Value, overflowValue(metric.Mib))

		var name string
		ifName := escapeInterfaceName(metric.IfName)
		if deltaValues(metric.Mib) {
			direction := "txBytes"
			if receiveDirection(metric.Mib) {
				direction = "rxBytes"
			}
			name = fmt.Sprintf("interface.%s.%s.delta", ifName, direction)
			value /= 60
		} else {
			name = fmt.Sprintf("custom.interface.%s.%s", metric.Mib, ifName)
		}
		metrics = append(metrics, &mackerel.MetricValue{
			Name:  name,
			Time:  now,
			Value: value,
		})
	}
	return metrics
}

func escapeInterfaceName(ifName string) string {
	return strings.Replace(strings.Replace(strings.Replace(ifName, "/", "-", -1), ".", "_", -1), " ", "", -1)
}

func overflowValue(mib string) uint64 {
	if mib == "ifInOctets" || mib == "ifOutOctets" {
		return math.MaxUint32
	}
	return math.MaxUint64
}

func receiveDirection(mib string) bool {
	return (mib == "ifInOctets" || mib == "ifHCInOctets")
}

func deltaValues(mib string) bool {
	return mib == "ifInOctets" || mib == "ifOutOctets" || mib == "ifHCInOctets" || mib == "ifHCOutOctets"
}

func calcurateDiff(a, b, overflow uint64) uint64 {
	if b < a {
		return overflow - a + b
	} else {
		return b - a
	}
}
