package collector

import (
	"context"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/tenntenn/golden"
	"github.com/tenntenn/testtime"
	"github.com/yseto/switch-traffic-to-mackerel/config"
)

var (
	flagUpdate bool
	goldenDir  string = "./testdata/"
)

func init() {
	flag.BoolVar(&flagUpdate, "update", true, "update golden files")
}

func TestDebugPrint(t *testing.T) {
	ctx := context.Background()

	got := capture(func() {
		testtime.SetTime(t, time.Unix(1, 0))

		c := &config.Config{
			MIBs:  []string{"ifHCInOctets", "ifHCOutOctets"},
			Debug: true,
		}
		_, err := do(ctx, &mockSnmpClient{}, c)
		if err != nil {
			t.Error("invalid raised error")
		}
	})

	if diff := golden.Check(t, flagUpdate, goldenDir, t.Name(), got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func capture(f func()) string {
	writer := os.Stdout
	defer func() {
		os.Stdout = writer
	}()

	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}
