package mackerel

import (
	"container/list"
	"context"
	"log"
	"sync"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
)

type mackerelClient interface {
	CreateHost(param *mackerel.CreateHostParam) (string, error)
	UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error)
	CreateGraphDefs(payloads []*mackerel.GraphDefsParam) error
	PostHostMetricValuesByHostID(hostID string, metricValues []*mackerel.MetricValue) error
}

type Queue struct {
	sync.Mutex

	buffers  *list.List
	Snapshot []collector.MetricsDutum
	client   mackerelClient

	hostID     string
	targetAddr string
	name       string
}

type QueueArg struct {
	Apikey     string
	HostID     string
	TargetAddr string
	Name       string
	Snapshot   []collector.MetricsDutum
}

func NewQueue(qa *QueueArg) *Queue {
	client := mackerel.NewClient(qa.Apikey)

	return &Queue{
		buffers:    list.New(),
		client:     client,
		Snapshot:   qa.Snapshot,
		hostID:     qa.HostID,
		targetAddr: qa.TargetAddr,
		name:       qa.Name,
	}
}

// return host ID when create.
func (q *Queue) Init(ifs []collector.Interface) (*string, error) {
	log.Println("init queue")

	var interfaces []mackerel.Interface

	if len(ifs) == 0 {
		interfaces = []mackerel.Interface{
			{
				Name:          "main",
				IPv4Addresses: []string{q.targetAddr},
			},
		}
	} else {
		for i := range ifs {
			interfaces = append(interfaces, mackerel.Interface{
				Name:          ifs[i].IfName,
				IPv4Addresses: ifs[i].IpAddress,
			})
		}
	}

	var newHostID *string
	var err error
	if q.hostID != "" {
		_, err = q.client.UpdateHost(q.hostID, &mackerel.UpdateHostParam{
			Name:       q.name,
			Interfaces: interfaces,
		})
	} else {
		q.hostID, err = q.client.CreateHost(&mackerel.CreateHostParam{
			Name:       q.name,
			Interfaces: interfaces,
		})
		newHostID = &q.hostID
	}
	if err != nil {
		return nil, err
	}

	err = q.client.CreateGraphDefs(graphDefs)
	if err != nil {
		return nil, err
	}
	return newHostID, nil
}

func (q *Queue) SendTicker(ctx context.Context, wg *sync.WaitGroup) {
	t := time.NewTicker(500 * time.Millisecond)

	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			q.sendToMackerel(ctx)

		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}

func (q *Queue) sendToMackerel(ctx context.Context) {
	if q.buffers.Len() == 0 {
		return
	}

	e := q.buffers.Front()
	err := q.client.PostHostMetricValuesByHostID(q.hostID, e.Value.([](*mackerel.MetricValue)))
	if err != nil {
		log.Println(err)
		return
	}
	q.Lock()
	q.buffers.Remove(e)
	q.Unlock()
}
