package snmp

import (
	"context"
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
