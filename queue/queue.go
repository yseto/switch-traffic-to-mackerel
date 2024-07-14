package queue

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/mackerelio/mackerel-client-go"
)

type SendInterface interface {
	Send(context.Context, []*mackerel.MetricValue) error
}

type Queue struct {
	sync.Mutex
	buffers *list.List

	sendFunc SendInterface

	debug  bool
	dryrun bool
}

type Arg struct {
	SendFunc SendInterface

	Debug  bool
	DryRun bool
}

type noopSendFunc struct{}

func (noopSendFunc) Send(_ context.Context, _ []*mackerel.MetricValue) error {
	return nil
}

func New(qa Arg) *Queue {
	if qa.SendFunc == nil {
		qa.SendFunc = &noopSendFunc{}
	}
	return &Queue{
		buffers: list.New(),

		sendFunc: qa.SendFunc,
		debug:    qa.Debug,
		dryrun:   qa.DryRun,
	}
}

func (q *Queue) Tick(ctx context.Context) {
	if q.buffers.Len() == 0 {
		return
	}

	e := q.buffers.Front()
	value := e.Value.([](*mackerel.MetricValue))

	if q.debug {
		for idx := range value {
			fmt.Printf("%d\t%s\t%v\n", value[idx].Time, value[idx].Name, value[idx].Value)
		}
	}

	if !q.dryrun {
		err := q.sendFunc.Send(ctx, value)
		if err != nil {
			log.Println(err)
			return
		}
	}

	q.Lock()
	q.buffers.Remove(e)
	q.Unlock()
}

func (q *Queue) Enqueue(rawMetrics []*mackerel.MetricValue) {
	q.Lock()
	q.buffers.PushBack(rawMetrics)
	q.Unlock()
}
