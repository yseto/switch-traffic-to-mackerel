package snmp

import (
	"context"
	"time"

	"github.com/gosnmp/gosnmp"
)

type Handler interface {
	Get(oids []string) (result *gosnmp.SnmpPacket, err error)
	BulkWalk(rootOid string, walkFn gosnmp.WalkFunc) error

	Connect() error
	Close() error
}

type snmpHandler struct {
	gosnmp.GoSNMP
}

func NewHandler(ctx context.Context, target, community string) Handler {
	return &snmpHandler{
		gosnmp.GoSNMP{
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
		},
	}
}

func (x *snmpHandler) Close() error {
	return x.GoSNMP.Conn.Close()
}
