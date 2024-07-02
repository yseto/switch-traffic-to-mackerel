package queue

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mackerelio/mackerel-client-go"
)

func TestNew(t *testing.T) {
	q := New(Arg{})
	q.sendFunc.Send(context.TODO(), nil) // nolint
}

type mockSendFunc struct {
	count  int
	values []*mackerel.MetricValue
}

func (m *mockSendFunc) Send(_ context.Context, v []*mackerel.MetricValue) error {
	m.count++
	m.values = append(m.values, v...)
	return nil
}

func TestTick(t *testing.T) {
	t.Run("empty queue", func(t *testing.T) {
		mock := &mockSendFunc{}
		q := New(Arg{
			SendFunc: mock,
		})

		q.Tick(context.TODO())

		if mock.count != 0 {
			t.Error("invalid. called Send()")
		}
	})

	t.Run("exist queue", func(t *testing.T) {
		tm := time.Now().Unix()
		mock := &mockSendFunc{}
		q := New(Arg{
			SendFunc: mock,
		})

		q.Enqueue([]*mackerel.MetricValue{
			{
				Name:  "name12345",
				Time:  tm,
				Value: 1.2345,
			},
		})
		q.Enqueue([]*mackerel.MetricValue{
			{
				Name:  "name12345678",
				Time:  tm,
				Value: 1.2345678,
			},
		})

		q.Tick(context.TODO())

		actual := mock.values
		expected := []*mackerel.MetricValue{
			{
				Name:  "name12345",
				Time:  tm,
				Value: 1.2345,
			},
		}
		if diff := cmp.Diff(actual, expected); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
		if mock.count != 1 {
			t.Error("invalid. called Send()")
		}

		q.Tick(context.TODO())

		actual = mock.values
		expected = append(expected, &mackerel.MetricValue{
			Name:  "name12345678",
			Time:  tm,
			Value: 1.2345678,
		})

		if diff := cmp.Diff(actual, expected); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
		if mock.count != 2 {
			t.Error("invalid. called Send()")
		}
	})
}
