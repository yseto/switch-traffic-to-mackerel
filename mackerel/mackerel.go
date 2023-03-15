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

type MackerelClient interface {
	CreateHost(param *mackerel.CreateHostParam) (string, error)
	UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error)
	CreateGraphDefs(payloads []*mackerel.GraphDefsParam) error
	PostHostMetricValuesByHostID(hostID string, metricValues []*mackerel.MetricValue) error
}

type Queue struct {
	sync.Mutex

	buffers  *list.List
	Snapshot []collector.MetricsDutum
	client   MackerelClient
}

func NewQueue(apikey string, snapshot []collector.MetricsDutum) *Queue {
	client := mackerel.NewClient(apikey)

	return &Queue{
		buffers:  list.New(),
		client:   client,
		Snapshot: snapshot,
	}
}

func (q *Queue) InitialForMackerel(c *config.Config) (*string, error) {
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
		_, err = q.client.UpdateHost(hostId, &mackerel.UpdateHostParam{
			Name:       c.Name,
			Interfaces: interfaces,
		})
		if err != nil {
			return nil, err
		}
	} else {
		hostId, err = q.client.CreateHost(&mackerel.CreateHostParam{
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
	err = q.client.CreateGraphDefs(GraphDefs)
	if err != nil {
		return nil, err
	}
	return &hostId, nil
}

func (q *Queue) SendTicker(ctx context.Context, wg *sync.WaitGroup, hostId *string) {
	t := time.NewTicker(500 * time.Millisecond)

	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			q.sendToMackerel(ctx, hostId)

		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}

func (q *Queue) sendToMackerel(ctx context.Context, hostId *string) {
	if q.buffers.Len() == 0 {
		return
	}

	e := q.buffers.Front()
	// log.Infof("send current value: %#v", e.Value)
	// log.Infof("buffers len: %d", buffers.Len())

	err := q.client.PostHostMetricValuesByHostID(*hostId, e.Value.([](*mackerel.MetricValue)))
	if err != nil {
		log.Println(err)
		return
	} else {
		log.Println("success")
	}
	q.Lock()
	q.buffers.Remove(e)
	q.Unlock()
}
