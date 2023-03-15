package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	mackerel "github.com/mackerelio/mackerel-client-go"
	"github.com/yseto/switch-traffic-to-mackerel/collector"
	"github.com/yseto/switch-traffic-to-mackerel/config"
	mckr "github.com/yseto/switch-traffic-to-mackerel/mackerel"
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
	mckr.Snapshot, err = collector.Do(ctx, collectParams)
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

	hostId, err := mckr.InitialForMackerel(collectParams, client)
	if err != nil {
		return err
	}

	wg.Add(1)
	go mckr.SendTicker(ctx, &wg, client, hostId)
	wg.Wait()

	return nil
}

func ticker(ctx context.Context, wg *sync.WaitGroup, collectParams *config.Config) {
	t := time.NewTicker(1 * time.Minute)
	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			rawMetrics, err := collector.Do(ctx, collectParams)
			if err != nil {
				log.Println(err.Error())
			}
			if !collectParams.DryRun {
				mckr.Enqueue(rawMetrics)
			}
		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}
