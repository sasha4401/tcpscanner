package tcpscanner

import (
	"context"
	"errors"
	"sync"
)

var ErrPoolClosed = errors.New("pool is closed")

type pool struct {
	workers int
	jobs    chan job
	results chan Result
	stop    chan struct{}
	done    chan struct{}

	wg       sync.WaitGroup
	once     sync.Once
	closeRes sync.Once
}

type job struct {
	host string
	port uint16
	ctx  context.Context
	ip   string
	fn   func(ctx context.Context, host string, port uint16, ip string) Result
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

func (p *pool) submit(ctx context.Context, host string, port uint16, ip string, fn func(ctx context.Context, host string, port uint16, ip string) Result) error {
	select {
	case <-p.stop:
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	j := job{ctx: ctx, host: host, port: port, fn: fn, ip: ip}
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
	res := j.fn(j.ctx, j.host, j.port, j.ip)
	p.sendRes(res)
}

func (p *pool) shutdown(ctx context.Context) error {
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
		p.closeRes.Do(func() { close(p.results) })
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
	select {
	case p.results <- res:
	case <-p.done:
	}
}

func (p *pool) readResults() <-chan Result {
	return p.results
}
