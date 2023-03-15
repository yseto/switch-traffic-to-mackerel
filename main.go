package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/yseto/switch-traffic-to-mackerel/collector"
	"github.com/yseto/switch-traffic-to-mackerel/config"
	"github.com/yseto/switch-traffic-to-mackerel/mackerel"
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

	c, err := config.Init(filename)
	if err != nil {
		log.Fatal(err)
	}
	c.Debug = (c.Debug || debug)
	c.DryRun = (c.DryRun || dryrun)

	if c.Mackerel == nil {
		log.Println("force dry-run.")
		c.DryRun = true
	}

	snapshot, err := collector.Do(ctx, c)
	if err != nil {
		log.Fatal(err)
	}

	qa := &mackerel.QueueArg{
		TargetAddr: c.Target,
		Snapshot:   snapshot,
	}
	if c.Mackerel != nil {
		qa.Apikey = c.Mackerel.ApiKey
		qa.HostID = c.Mackerel.HostID
		qa.Name = c.Mackerel.Name
		if qa.Name == "" {
			qa.Name = c.Target
		}
	}
	queue := mackerel.NewQueue(qa)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go ticker(ctx, wg, c, queue)

	if c.DryRun {
		wg.Wait()
		return
	}

	newHostID, err := queue.Init()
	if err != nil {
		log.Fatal(err)
	}
	if newHostID != nil {
		log.Println("save HostID")
		if err = c.Save(*newHostID); err != nil {
			log.Fatal(err)
		}
	}

	wg.Add(1)
	go queue.SendTicker(ctx, wg)
	wg.Wait()
}

func ticker(ctx context.Context, wg *sync.WaitGroup, c *config.Config, queue *mackerel.Queue) {
	t := time.NewTicker(1 * time.Minute)
	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			rawMetrics, err := collector.Do(ctx, c)
			if err != nil {
				log.Println(err.Error())
			}
			if !c.DryRun {
				queue.Enqueue(rawMetrics)
			}
		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}
