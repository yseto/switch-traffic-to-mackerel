package mackerel

import (
	"context"
	"log"

	mackerel "github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
)

type mackerelClient interface {
	CreateHost(param *mackerel.CreateHostParam) (string, error)
	UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error)
	CreateGraphDefs(payloads []*mackerel.GraphDefsParam) error
	PostHostMetricValuesByHostID(hostID string, metricValues []*mackerel.MetricValue) error
}

type Mackerel struct {
	client     mackerelClient
	hostID     string
	targetAddr string
	name       string
}

type Arg struct {
	Apikey     string
	HostID     string
	TargetAddr string
	Name       string
}

func New(qa *Arg) *Mackerel {
	client := mackerel.NewClient(qa.Apikey)

	return &Mackerel{
		client:     client,
		hostID:     qa.HostID,
		targetAddr: qa.TargetAddr,
		name:       qa.Name,
	}
}

// return host ID when create.
func (m *Mackerel) Init(ifs []collector.Interface) (*string, error) {
	log.Println("init mackerel")

	var interfaces []mackerel.Interface

	if len(ifs) == 0 {
		interfaces = []mackerel.Interface{
			{
				Name:          "main",
				IPv4Addresses: []string{m.targetAddr},
			},
		}
	} else {
		for i := range ifs {
			interfaces = append(interfaces, mackerel.Interface{
				Name:          ifs[i].IfName,
				IPv4Addresses: ifs[i].IpAddress,
				MacAddress:    ifs[i].MacAddress,
			})
		}
	}

	var newHostID *string
	var err error
	if m.hostID != "" {
		_, err = m.client.UpdateHost(m.hostID, &mackerel.UpdateHostParam{
			Name:       m.name,
			Interfaces: interfaces,
		})
	} else {
		m.hostID, err = m.client.CreateHost(&mackerel.CreateHostParam{
			Name:       m.name,
			Interfaces: interfaces,
		})
		newHostID = &m.hostID
	}
	if err != nil {
		return nil, err
	}

	if err = m.CreateGraphDefs(graphDefs); err != nil {
		return nil, err
	}
	return newHostID, nil
}

func (m *Mackerel) CreateGraphDefs(d []*mackerel.GraphDefsParam) error {
	return m.client.CreateGraphDefs(d)
}

func (m *Mackerel) Send(ctx context.Context, value []*mackerel.MetricValue) error {
	return m.client.PostHostMetricValuesByHostID(m.hostID, value)
}
