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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var filename string
	var debug, dryrun bool
	flag.StringVar(&filename, "config", "config.yaml", "config `filename`")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.BoolVar(&dryrun, "dry-run", false, "dry run")
	flag.Parse()

	collectParams, err := config.Init(filename)
	if err != nil {
		log.Fatal(err)
	}
	collectParams.Debug = (collectParams.Debug || debug)
	collectParams.DryRun = (collectParams.DryRun || dryrun)

	log.Println("start")

	if collectParams.Mackerel == nil {
		_, err := collector.Do(ctx, collectParams)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = runMackerel(ctx, collectParams)
		if err != nil {
			log.Fatal(err)
		}
	}
}
