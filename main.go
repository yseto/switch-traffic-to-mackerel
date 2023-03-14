package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
	"github.com/yseto/switch-traffic-to-mackerel/config"
)

var log = logrus.New()
var apikey = os.Getenv("MACKEREL_API_KEY")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	var filename string
	flag.StringVar(&filename, "config", "config.yaml", "config `filename`")
	flag.Parse()

	collectParams, err := config.Parse(filename)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("start")

	if apikey == "" {
		_, err := collector.Do(ctx, collectParams)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		runMackerel(ctx, collectParams)
	}
}
