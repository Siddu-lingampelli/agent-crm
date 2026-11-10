// Package jobs provides a bounded in-process worker pool.
package jobs

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
)

type Config struct {
	Workers   int
	QueueSize int
}
type job struct {
	name string
	run  func(context.Context)
}
type Pool struct {
	cfg        Config
	log        *slog.Logger
	mu         sync.Mutex
	closed     bool
	queue      chan job
	done       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	processing atomic.Int64
	completed  atomic.Int64
	failed     atomic.Int64
}

func New(cfg Config, log *slog.Logger) *Pool {
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 256
	}
	if log == nil {
		log = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{cfg: cfg, log: log, queue: make(chan job, cfg.QueueSize), done: make(chan struct{}), ctx: ctx, cancel: cancel}
	var workers sync.WaitGroup
	for range cfg.Workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for task := range p.queue {
				if p.ctx.Err() != nil {
					continue
				}
				p.run(task)
			}
		}()
	}
	go func() { workers.Wait(); close(p.done) }()
	return p
}
func (p *Pool) run(task job) {
	p.processing.Add(1)
	defer p.processing.Add(-1)
	defer func() {
		if recovered := recover(); recovered != nil {
			p.failed.Add(1)
			p.log.Error("background job panicked", "name", task.name)
		}
	}()
	task.run(p.ctx)
	p.completed.Add(1)
}
func (p *Pool) Enqueue(ctx context.Context, name string, run func(context.Context)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if run == nil {
		return errors.New("job callback required")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.New("job pool is closed")
	}
	select {
	case p.queue <- job{name, run}:
		return nil
	default:
		return errors.New("job queue is full")
	}
}
func (p *Pool) Submit(ctx context.Context, name string) {
	_ = p.Enqueue(ctx, name, func(context.Context) {})
}
func (p *Pool) Status() map[string]any {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	return map[string]any{"workers": p.cfg.Workers, "queue_size": p.cfg.QueueSize, "closed": closed, "pending": len(p.queue), "processing": p.processing.Load(), "completed": p.completed.Load(), "failed": p.failed.Load()}
}
func (p *Pool) Shutdown(ctx context.Context) {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.queue)
	}
	p.mu.Unlock()
	select {
	case <-p.done:
	case <-ctx.Done():
		p.cancel()
	}
}
func (p *Pool) Close() error { p.cancel(); p.Shutdown(context.Background()); return nil }
