package jobs

import (
	"context"
	"testing"
	"time"
)

func TestPoolBoundsQueueAndCancelsWork(t *testing.T) {
	pool := New(Config{Workers: 1, QueueSize: 1}, nil)
	started := make(chan struct{})
	if err := pool.Enqueue(context.Background(), "blocking", func(ctx context.Context) { close(started); <-ctx.Done() }); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := pool.Enqueue(context.Background(), "queued", func(context.Context) {}); err != nil {
		t.Fatal(err)
	}
	if err := pool.Enqueue(context.Background(), "overflow", func(context.Context) {}); err == nil {
		t.Fatal("accepted queue overflow")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	pool.Shutdown(ctx)
	pool.Close()
	if err := pool.Enqueue(context.Background(), "closed", func(context.Context) {}); err == nil {
		t.Fatal("accepted work after close")
	}
}
