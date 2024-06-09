package queue

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	q := New(Arg{})
	q.sendFunc.Send(context.TODO(), nil) // nolint
}
