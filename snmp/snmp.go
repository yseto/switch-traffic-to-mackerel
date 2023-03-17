package snmp

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

const (
	MIBifNumber     = "1.3.6.1.2.1.2.1.0"
	MIBifDescr      = "1.3.6.1.2.1.2.2.1.2"
	MIBifOperStatus = "1.3.6.1.2.1.2.2.1.8"
)

type SNMP struct {
	handler Handler
}

func Init(ctx context.Context, target, community string) (*SNMP, error) {
	g := NewHandler(ctx, target, community)
	err := g.Connect()
	if err != nil {
		return nil, err
	}
	return &SNMP{handler: g}, nil
}

func (s *SNMP) Close() error {
	return s.handler.Close()
}

var (
	errGetInterfaceNumber = errors.New("cant get interface number")
	errParseInterfaceName = errors.New("cant parse interface name")
	errParseError         = errors.New("cant parse value.")
)

func (s *SNMP) GetInterfaceNumber() (uint64, error) {
	result, err := s.handler.Get([]string{MIBifNumber})
	if err != nil {
		return 0, err
	}
	variable := result.Variables[0]
	switch variable.Type {
	case gosnmp.OctetString:
		return 0, errGetInterfaceNumber
	default:
		return gosnmp.ToBigInt(variable.Value).Uint64(), nil
	}
}

func (s *SNMP) BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error) {
	kv := make(map[uint64]string, length)
	err := s.handler.BulkWalk(MIBifDescr, func(pdu gosnmp.SnmpPDU) error {
		index, err := captureIfIndex(pdu.Name)
		if err != nil {
			return err
		}
		switch pdu.Type {
		case gosnmp.OctetString:
			kv[index] = string(pdu.Value.([]byte))
		default:
			return errParseInterfaceName
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

func (s *SNMP) BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error) {
	kv := make(map[uint64]bool, length)
	err := s.handler.BulkWalk(MIBifOperStatus, func(pdu gosnmp.SnmpPDU) error {
		index, err := captureIfIndex(pdu.Name)
		if err != nil {
			return err
		}
		/*
			up(1)
			down(2)
			testing(3)
		*/
		switch pdu.Type {
		case gosnmp.OctetString:
			return errParseError
		default:
			tmp := gosnmp.ToBigInt(pdu.Value).Uint64()
			if tmp != 2 {
				kv[index] = true
			} else {
				kv[index] = false
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

func (s *SNMP) BulkWalk(oid string, length uint64) (map[uint64]uint64, error) {
	kv := make(map[uint64]uint64, length)
	err := s.handler.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		index, err := captureIfIndex(pdu.Name)
		if err != nil {
			return err
		}
		switch pdu.Type {
		case gosnmp.OctetString:
			return errParseError
		default:
			kv[index] = gosnmp.ToBigInt(pdu.Value).Uint64()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

func captureIfIndex(name string) (uint64, error) {
	sl := strings.Split(name, ".")
	return strconv.ParseUint(sl[len(sl)-1], 10, 64)
}
