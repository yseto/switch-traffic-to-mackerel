package main

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
	"github.com/yseto/switch-traffic-to-mackerel/config"
	mckr "github.com/yseto/switch-traffic-to-mackerel/mackerel"
)

var buffers = list.New()
var mutex = &sync.Mutex{}
var snapshot []collector.MetricsDutum

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

func runMackerel(ctx context.Context, collectParams *config.Config) error {
	var err error
	snapshot, err = collector.Do(ctx, collectParams)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go ticker(ctx, &wg, collectParams)

	if collectParams.DryRun {
		wg.Wait()
		return nil
	}

	client := mackerel.NewClient(collectParams.Mackerel.ApiKey)

	hostId, err := initialForMackerel(collectParams, client)
	if err != nil {
		return err
	}

	wg.Add(1)
	go sendTicker(ctx, &wg, client, hostId)
	wg.Wait()

	return nil
}

func initialForMackerel(c *config.Config, client *mackerel.Client) (*string, error) {
	log.Println("init for mackerel")

	idPath, err := c.HostIdPath()
	if err != nil {
		return nil, err
	}
	interfaces := []mackerel.Interface{
		{
			Name:          "main",
			IPv4Addresses: []string{c.Target},
		},
	}
	var hostId string
	if _, err := os.Stat(idPath); err == nil {
		bytes, err := os.ReadFile(idPath)
		if err != nil {
			return nil, err
		}
		hostId = string(bytes)
		_, err = client.UpdateHost(hostId, &mackerel.UpdateHostParam{
			Name:       c.Name,
			Interfaces: interfaces,
		})
		if err != nil {
			return nil, err
		}
	} else {
		hostId, err = client.CreateHost(&mackerel.CreateHostParam{
			Name:       c.Name,
			Interfaces: interfaces,
		})
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(idPath, []byte(hostId), 0666)
		if err != nil {
			return nil, err
		}
	}
	err = client.CreateGraphDefs(mckr.GraphDefs)
	if err != nil {
		return nil, err
	}
	return &hostId, nil
}

func ticker(ctx context.Context, wg *sync.WaitGroup, collectParams *config.Config) {
	t := time.NewTicker(1 * time.Minute)
	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			err := innerTicker(ctx, collectParams)
			if err != nil {
				log.Println(err.Error())
			}
		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
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

func innerTicker(ctx context.Context, collectParams *config.Config) error {
	rawMetrics, err := collector.Do(ctx, collectParams)
	if err != nil {
		return err
	}

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

	return nil
}

func sendTicker(ctx context.Context, wg *sync.WaitGroup, client *mackerel.Client, hostId *string) {
	t := time.NewTicker(500 * time.Millisecond)

	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			sendToMackerel(ctx, client, hostId)

		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}

func sendToMackerel(ctx context.Context, client *mackerel.Client, hostId *string) {
	if buffers.Len() == 0 {
		return
	}

	e := buffers.Front()
	// log.Infof("send current value: %#v", e.Value)
	// log.Infof("buffers len: %d", buffers.Len())

	err := client.PostHostMetricValuesByHostID(*hostId, e.Value.([](*mackerel.MetricValue)))
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Println("success")
	}
	mutex.Lock()
	buffers.Remove(e)
	mutex.Unlock()
}
