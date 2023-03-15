package mackerel

import (
	"container/list"
	"context"
	"log"
	"os"
	"sync"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
	"github.com/yseto/switch-traffic-to-mackerel/config"
)

var buffers = list.New()
var mutex = &sync.Mutex{}
var Snapshot []collector.MetricsDutum

func InitialForMackerel(c *config.Config, client *mackerel.Client) (*string, error) {
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
	err = client.CreateGraphDefs(GraphDefs)
	if err != nil {
		return nil, err
	}
	return &hostId, nil
}

func SendTicker(ctx context.Context, wg *sync.WaitGroup, client *mackerel.Client, hostId *string) {
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
