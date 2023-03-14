package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
	"github.com/yseto/switch-traffic-to-mackerel/config"
)

var apikey = os.Getenv("MACKEREL_API_KEY")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var filename string
	var debug bool
	flag.StringVar(&filename, "config", "config.yaml", "config `filename`")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.Parse()

	collectParams, err := config.Parse(filename)
	if err != nil {
		log.Fatal(err)
	}
	collectParams.Debug = (collectParams.Debug || debug)

	log.Println("start")

	if apikey == "" {
		_, err := collector.Do(ctx, collectParams)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		runMackerel(ctx, collectParams)
	}
}
