package snmp

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gosnmp/gosnmp"
)

type mockHandler struct {
	oids    []string
	rootOid string
	result  *gosnmp.SnmpPacket
	pdus    []gosnmp.SnmpPDU
}

func (m *mockHandler) Get(oids []string) (result *gosnmp.SnmpPacket, err error) {
	m.oids = oids
	return m.result, nil
}

func (m *mockHandler) BulkWalk(rootOid string, walkFn gosnmp.WalkFunc) error {
	m.rootOid = rootOid
	for i := range m.pdus {
		if err := walkFn(m.pdus[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockHandler) Connect() error {
	return nil
}

func (m *mockHandler) Close() error {
	return nil
}

func TestGetInterfaceNumber(t *testing.T) {
	m := mockHandler{
		result: &gosnmp.SnmpPacket{
			Variables: []gosnmp.SnmpPDU{
				{
					Value: uint64(3),
				},
			},
		},
	}
	s := &SNMP{handler: &m}

	actual, err := s.GetInterfaceNumber()
	if err != nil {
		t.Error("failed raised error")
	}
	if actual != 3 {
		t.Error("invalid result")
	}
	if !reflect.DeepEqual(m.oids, []string{MIBifNumber}) {
		t.Error("invalid argument")
	}
}

func TestBulkWalkGetInterfaceName(t *testing.T) {
	m := mockHandler{
		pdus: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.2.1",
				Value: []byte("lo0"),
				Type:  gosnmp.OctetString,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.2.2",
				Value: []byte("eth0"),
				Type:  gosnmp.OctetString,
			},
		},
	}
	s := &SNMP{handler: &m}

	actual, err := s.BulkWalkGetInterfaceName(2)
	expected := map[uint64]string{
		1: "lo0",
		2: "eth0",
	}
	if err != nil {
		t.Error("failed raised error")
	}
	if d := cmp.Diff(actual, expected); d != "" {
		t.Error("invalid result")
	}
	if !reflect.DeepEqual(m.rootOid, MIBifDescr) {
		t.Error("invalid argument")
	}
}

func TestBulkWalkGetInterfaceState(t *testing.T) {
	m := mockHandler{
		pdus: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.8.1",
				Value: 1,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.8.2",
				Value: 2,
			},
		},
	}
	s := &SNMP{handler: &m}

	actual, err := s.BulkWalkGetInterfaceState(2)
	expected := map[uint64]bool{
		1: true,
		2: false,
	}
	if err != nil {
		t.Error("failed raised error")
	}
	if d := cmp.Diff(actual, expected); d != "" {
		t.Error("invalid result")
	}
	if !reflect.DeepEqual(m.rootOid, MIBifOperStatus) {
		t.Error("invalid argument")
	}
}

func TestBulkWalk(t *testing.T) {
	m := mockHandler{
		pdus: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.2.2.1.10.1",
				Value: 1,
			},
			{
				Name:  "1.3.6.1.2.1.2.2.1.10.2",
				Value: 2,
			},
		},
	}
	s := &SNMP{handler: &m}

	actual, err := s.BulkWalk("1.3.6.1.2.1.2.2.1.10", 2)
	expected := map[uint64]uint64{
		1: 1,
		2: 2,
	}
	if err != nil {
		t.Error("failed raised error")
	}
	if d := cmp.Diff(actual, expected); d != "" {
		t.Error("invalid result")
	}
	if !reflect.DeepEqual(m.rootOid, "1.3.6.1.2.1.2.2.1.10") {
		t.Error("invalid argument")
	}
}

func TestBulkWalkGetInterfaceIPAddress(t *testing.T) {
	m := mockHandler{
		pdus: []gosnmp.SnmpPDU{
			{
				Name:  "1.3.6.1.2.1.4.20.1.2.192.0.2.1",
				Value: 1,
			},
			{
				Name:  "1.3.6.1.2.1.4.20.1.2.192.0.2.2",
				Value: 1,
			},
			// invalid ip
			{
				Name:  "1.3.6.1.2.1.4.20.1.2.1024.1.2.3",
				Value: 2,
			},
			{
				Name:  "1.3.6.1.2.1.4.20.1.2.198.51.100.1",
				Value: 3,
			},
			{
				Name:  "1.3.6.1.2.1.4.20.1.2.127.0.0.1",
				Value: 4,
			},
		},
	}
	s := &SNMP{handler: &m}

	actual, err := s.BulkWalkGetInterfaceIPAddress()
	expected := map[uint64][]string{
		1: {"192.0.2.1", "192.0.2.2"},
		3: {"198.51.100.1"},
	}
	if err != nil {
		t.Error("failed raised error")
	}
	if d := cmp.Diff(actual, expected); d != "" {
		t.Errorf("invalid result %s", d)
	}
}
