package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"

	mackerel "github.com/mackerelio/mackerel-client-go"
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

	err = run(ctx, collectParams)
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, collectParams *config.Config) error {
	var err error
	snapshot, err = collector.Do(ctx, collectParams)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go ticker(ctx, &wg, collectParams)

	if collectParams.DryRun {
		wg.Wait()
		return nil
	}

	client := mackerel.NewClient(collectParams.Mackerel.ApiKey)

	hostId, err := initialForMackerel(collectParams, client)
	if err != nil {
		return err
	}

	wg.Add(1)
	go sendTicker(ctx, &wg, client, hostId)
	wg.Wait()

	return nil
}
