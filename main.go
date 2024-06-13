package main

import (
	"cmp"
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
	"github.com/yseto/switch-traffic-to-mackerel/queue"
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

	var mClient *mackerel.Mackerel
	if c.Mackerel != nil {
		mClient = mackerel.New(&mackerel.Arg{
			TargetAddr: c.Target,
			Apikey:     c.Mackerel.ApiKey,
			HostID:     c.Mackerel.HostID,
			Name:       cmp.Or(c.Mackerel.Name, c.Target),
		})

		var interfaces []collector.Interface
		if !c.Mackerel.IgnoreNetworkInfo {
			interfaces, err = collector.DoInterfaceIPAddress(ctx, c)
			if err != nil {
				log.Println("HINT: try mackerel > ignore-network-info: true")
				log.Fatal(err)
			}
		}

		newHostID, err := mClient.Init(interfaces)
		if err != nil {
			log.Fatal(err)
		}
		if newHostID != nil {
			log.Println("save HostID")
			if err = c.Save(*newHostID); err != nil {
				log.Fatal(err)
			}
		}
		if len(c.CustomMIBsGraphDefs) > 0 {
			if err = mClient.CreateGraphDefs(c.CustomMIBsGraphDefs); err != nil {
				log.Fatal(err)
			}
		}
	}

	queueHandler := queue.New(queue.Arg{
		SendFunc: mClient,
		Debug:    c.Debug,
		DryRun:   c.DryRun,
		Snapshot: snapshot,
	})

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go collectTicker(ctx, wg, c, queueHandler)

	wg.Add(1)
	go sendTicker(ctx, wg, queueHandler)
	wg.Wait()
}

func collectTicker(ctx context.Context, wg *sync.WaitGroup, c *config.Config, queueHandler *queue.Queue) {
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
				continue
			}
			queueHandler.Enqueue(rawMetrics)

			resp, err := collector.DoCustomMIBs(ctx, c)
			if err != nil {
				log.Println(err.Error())
				continue
			}

			bucket := make([]queue.CustomMIBValue, 0)
			for metricName, mib := range c.CustomMIBmetricNameMappedMIBs {
				if f, ok := resp[mib]; ok {
					bucket = append(bucket, queue.CustomMIBValue{
						Name:  metricName,
						Value: f,
					})
				}
			}
			queueHandler.EnqueueCustomMIB(bucket)

		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}

type sendTickerFunc interface {
	Tick(context.Context)
}

func sendTicker(ctx context.Context, wg *sync.WaitGroup, f sendTickerFunc) {
	t := time.NewTicker(500 * time.Millisecond)

	defer func() {
		t.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-t.C:
			f.Tick(ctx)

		case <-ctx.Done():
			log.Println("cancellation from context:", ctx.Err())
			return
		}
	}
}
