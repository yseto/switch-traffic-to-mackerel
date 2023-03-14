package snmp

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

var Default *gosnmp.GoSNMP

func Init(ctx context.Context, target, community string) {
	Default = &gosnmp.GoSNMP{
		Context:            ctx,
		Target:             target,
		Port:               161,
		Transport:          "udp",
		Community:          community,
		Version:            gosnmp.Version2c,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            gosnmp.MaxOids,
	}
}

const (
	MIBifNumber     = "1.3.6.1.2.1.2.1.0"
	MIBifDescr      = "1.3.6.1.2.1.2.2.1.2"
	MIBifOperStatus = "1.3.6.1.2.1.2.2.1.8"
)

func GetInterfaceNumber() (uint64, error) {
	result, err := Default.Get([]string{MIBifNumber})
	if err != nil {
		return 0, err
	}
	variable := result.Variables[0]
	switch variable.Type {
	case gosnmp.OctetString:
		return 0, errors.New("cant get interface number")
	default:
		return gosnmp.ToBigInt(variable.Value).Uint64(), nil
	}
}

func BulkWalkGetInterfaceName(length uint64) (map[uint64]string, error) {
	kv := make(map[uint64]string, length)
	err := Default.BulkWalk(MIBifDescr, func(pdu gosnmp.SnmpPDU) error {
		index, err := captureIfIndex(MIBifDescr, pdu.Name)
		if err != nil {
			return err
		}
		switch pdu.Type {
		case gosnmp.OctetString:
			kv[index] = string(pdu.Value.([]byte))
		default:
			return errors.New("cant parse interface name.")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return kv, nil
}

func BulkWalkGetInterfaceState(length uint64) (map[uint64]bool, error) {
	kv := make(map[uint64]bool, length)
	err := Default.BulkWalk(MIBifOperStatus, func(pdu gosnmp.SnmpPDU) error {
		index, err := captureIfIndex(MIBifOperStatus, pdu.Name)
		if err != nil {
			return err
		}
		switch pdu.Type {
		case gosnmp.OctetString:
			return errors.New("cant parse value.")
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

func BulkWalk(oid string, length uint64) (map[uint64]uint64, error) {
	kv := make(map[uint64]uint64, length)
	err := Default.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		index, err := captureIfIndex(oid, pdu.Name)
		if err != nil {
			return err
		}
		switch pdu.Type {
		case gosnmp.OctetString:
			return errors.New("cant parse value.")
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

func captureIfIndex(oid, name string) (uint64, error) {
	indexStr := strings.Replace(name, "."+oid+".", "", 1)
	return strconv.ParseUint(indexStr, 10, 64)
}
