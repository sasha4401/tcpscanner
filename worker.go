package tcpscanner

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

var ErrPoolClosed = errors.New("pool is closed")

type pool struct {
	workers int
	jobs    chan job
	results chan Result
	stop    chan struct{}
	done    chan struct{}

	wg   sync.WaitGroup
	once sync.Once
}

type job struct {
	host   string
	ip     net.IP
	port   uint16
	ctx    context.Context
	cancel context.CancelFunc
	fn     func(ctx context.Context, host string, port uint16) Result
}

func newPool(size int) *pool {
	p := &pool{
		workers: size,
		jobs:    make(chan job, 10*size),
		results: make(chan Result, 10*size),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}

	p.wg.Add(p.workers)
	for range p.workers {
		go p.worker()
	}

	return p
}

func (p *pool) Submit(ctx context.Context, host string, port uint16, fn func(ctx context.Context, host string, port uint16) Result, cancel context.CancelFunc) error {
	select {
	case <-p.stop:
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	j := job{ctx: ctx, host: host, port: port, fn: fn, cancel: cancel}
	select {
	case p.jobs <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.stop:
		return ErrPoolClosed
	}
}

func (p *pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.done:
			return
		case j := <-p.jobs:
			p.runJob(j)
		case <-p.stop:
			p.drainAndExit()
			return
		}
	}
}

func (p *pool) drainAndExit() {
	for {
		select {
		case <-p.done:
			return
		case j := <-p.jobs:
			p.runJob(j)
		default:
			return
		}
	}
}

func (p *pool) runJob(j job) {
	defer j.cancel()
	if err := j.ctx.Err(); err != nil {
		state := StateCanceled

		if errors.Is(err, context.DeadlineExceeded) {
			state = StateTimeout
		}

		p.sendRes(Result{
			Host:     j.host,
			IP:       j.ip,
			Port:     j.port,
			State:    state,
			Duration: 0 * time.Second,
			Err:      nil})

		return
	}

	res := j.fn(j.ctx, j.host, j.port)
	p.sendRes(res)
}

func (p *pool) Shutdown(ctx context.Context) error {
	called := false
	p.once.Do(func() {
		called = true
		close(p.stop)
	})
	if !called {
		return nil
	}

	drained := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(drained)
	}()

	select {
	case <-drained:
		return nil
	case <-ctx.Done():
		close(p.done)
		<-drained
		return ctx.Err()
	}
}

func (p *pool) sendRes(res Result) {
	p.results <- res
}

func (p *pool) Results() <-chan Result {
	return p.results
}
