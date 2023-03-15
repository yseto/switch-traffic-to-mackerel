package mackerel

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-client-go"
	"github.com/yseto/switch-traffic-to-mackerel/collector"
)

var overflowValue = map[string]uint64{
	"ifInOctets":    math.MaxUint32,
	"ifOutOctets":   math.MaxUint32,
	"ifHCInOctets":  math.MaxUint64,
	"ifHCOutOctets": math.MaxUint64,
	"ifInDiscards":  math.MaxUint64,
	"ifOutDiscards": math.MaxUint64,
	"ifInErrors":    math.MaxUint64,
	"ifOutErrors":   math.MaxUint64,
}

var receiveDirection = map[string]bool{
	"ifInOctets":   true,
	"ifHCInOctets": true,
	"ifInDiscards": true,
	"ifInErrors":   true,
}

var deltaValues = map[string]bool{
	"ifInOctets":    true,
	"ifOutOctets":   true,
	"ifHCInOctets":  true,
	"ifHCOutOctets": true,
}

func escapeInterfaceName(ifName string) string {
	return strings.Replace(strings.Replace(strings.Replace(ifName, "/", "-", -1), ".", "_", -1), " ", "", -1)
}

func calcurateDiff(a, b, overflow uint64) uint64 {
	if b < a {
		return overflow - a + b
	} else {
		return b - a
	}
}
func Enqueue(rawMetrics []collector.MetricsDutum) {
	prevSnapshot := snapshot
	snapshot = rawMetrics

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

		value := calcurateDiff(prevValue, metric.Value, overflowValue[metric.Mib])

		var name string
		ifName := escapeInterfaceName(metric.IfName)
		if deltaValues[metric.Mib] {
			direction := "txBytes"
			if receiveDirection[metric.Mib] {
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

	mutex.Lock()
	buffers.PushBack(metrics)
	mutex.Unlock()
}
