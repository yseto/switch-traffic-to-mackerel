package mackerel

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/mackerelio/mackerel-client-go"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
)

type mackerelClientMock struct {
	createParam  mackerel.CreateHostParam
	updateParam  mackerel.UpdateHostParam
	graphDef     []*mackerel.GraphDefsParam
	hostID       string
	metricValues []*mackerel.MetricValue

	returnHostID        string
	returnError         error
	returnErrorGraphDef error
}

func (m *mackerelClientMock) CreateHost(param *mackerel.CreateHostParam) (string, error) {
	m.createParam = *param
	return m.returnHostID, m.returnError
}
func (m *mackerelClientMock) UpdateHost(hostID string, param *mackerel.UpdateHostParam) (string, error) {
	m.updateParam = *param
	return m.returnHostID, m.returnError
}
func (m *mackerelClientMock) CreateGraphDefs(payloads []*mackerel.GraphDefsParam) error {
	m.graphDef = payloads
	return m.returnErrorGraphDef
}
func (m *mackerelClientMock) PostHostMetricValuesByHostID(hostID string, metricValues []*mackerel.MetricValue) error {
	m.hostID = hostID
	m.metricValues = metricValues
	return m.returnError
}

func TestInit(t *testing.T) {
	id := "1234567890"
	createHost := mackerel.CreateHostParam{
		Name: "hostname",
		Interfaces: []mackerel.Interface{
			{
				Name:          "main",
				IPv4Addresses: []string{"192.0.2.1"},
			},
		},
	}
	updateHost := mackerel.UpdateHostParam{
		Name: "hostname",
		Interfaces: []mackerel.Interface{
			{
				Name:          "main",
				IPv4Addresses: []string{"192.0.2.2"},
			},
		},
	}
	e := errors.New("error")
	tests := []struct {
		name                string
		expectedCreateParam mackerel.CreateHostParam
		expectedUpdateParam mackerel.UpdateHostParam
		expectedError       error
		expectedGraphDef    []*mackerel.GraphDefsParam
		hostID              string
		returnHostID        *string
		queue               *Mackerel
		mock                *mackerelClientMock
		interfaces          []collector.Interface
	}{
		{
			name:                "create host when hostID is empty",
			expectedCreateParam: createHost,
			queue: &Mackerel{
				name:       "hostname",
				targetAddr: "192.0.2.1",
			},
			returnHostID: &id,
			mock: &mackerelClientMock{
				returnHostID: "1234567890",
			},
			expectedGraphDef: graphDefs,
		},
		{
			name:                "update host when hostID is exist",
			expectedUpdateParam: updateHost,
			queue: &Mackerel{
				name:       "hostname",
				targetAddr: "192.0.2.2",
				hostID:     "0987654321",
			},
			mock:             &mackerelClientMock{},
			expectedGraphDef: graphDefs,
		},
		{
			name:                "create host is error",
			expectedCreateParam: createHost,
			expectedError:       e,
			queue: &Mackerel{
				name:       "hostname",
				targetAddr: "192.0.2.1",
			},
			mock: &mackerelClientMock{
				returnError: e,
			},
			expectedGraphDef: nil,
		},
		{
			name:                "update host is error",
			expectedUpdateParam: updateHost,
			expectedError:       e,
			queue: &Mackerel{
				name:       "hostname",
				targetAddr: "192.0.2.2",
				hostID:     "0987654321",
			},
			mock: &mackerelClientMock{
				returnError: e,
			},
			expectedGraphDef: nil,
		},
		{
			name:                "createGraphDef is error",
			expectedUpdateParam: updateHost,
			expectedError:       e,
			queue: &Mackerel{
				name:       "hostname",
				targetAddr: "192.0.2.2",
				hostID:     "0987654321",
			},
			mock: &mackerelClientMock{
				returnErrorGraphDef: e,
			},
			expectedGraphDef: graphDefs,
		},
		{
			name: "[]collector.interface is exist",
			expectedCreateParam: mackerel.CreateHostParam{
				Name: "hostname",
				Interfaces: []mackerel.Interface{
					{
						Name:          "eth0",
						IPv4Addresses: []string{"192.0.2.1", "192.0.2.2"},
					},
					{
						Name:          "eth1",
						IPv4Addresses: []string{"192.0.2.3"},
					},
				},
			},
			queue: &Mackerel{
				name:       "hostname",
				targetAddr: "192.0.2.1",
			},
			returnHostID: &id,
			mock: &mackerelClientMock{
				returnHostID: "1234567890",
			},
			expectedGraphDef: graphDefs,
			interfaces: []collector.Interface{
				{
					IfName:    "eth0",
					IpAddress: []string{"192.0.2.1", "192.0.2.2"},
				},
				{
					IfName:    "eth1",
					IpAddress: []string{"192.0.2.3"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.queue.client = tc.mock
			newHostID, err := tc.queue.Init(tc.interfaces)
			if !errors.Is(err, tc.expectedError) {
				t.Error("invalid error")
			}
			if !reflect.DeepEqual(newHostID, tc.returnHostID) {
				t.Error("newHostID is invalid")
			}
			if !reflect.DeepEqual(tc.mock.createParam, tc.expectedCreateParam) {
				t.Error("createParam is invalid")
			}
			if !reflect.DeepEqual(tc.mock.updateParam, tc.expectedUpdateParam) {
				t.Error("updateParam is invalid")
			}
			if !reflect.DeepEqual(tc.mock.graphDef, tc.expectedGraphDef) {
				t.Error("CreateGraphDefs is invalid")
			}
		})
	}

}

func TestSend(t *testing.T) {
	ctx := context.Background()

	mock := &mackerelClientMock{}
	mc := &Mackerel{
		hostID: "0987654321",
		client: mock,
	}

	if err := mc.Send(ctx, nil); err != nil {
		t.Errorf("occur error %v", err)
	}

	if mock.hostID == "" {
		t.Error("invalid need hostID")
	}

}
