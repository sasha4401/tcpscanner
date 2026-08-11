package tcpscanner

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_Concurrency(t *testing.T) {
	const concurrency = 3
	const jobs = 20

	p := newPool(concurrency)

	var running atomic.Int32
	var maxRunning atomic.Int32

	fn := func(ctx context.Context, host string, port uint16) Result {
		current := running.Add(1)

		for {
			old := maxRunning.Load()
			if current <= old || maxRunning.CompareAndSwap(old, current) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)

		running.Add(-1)

		return Result{
			Host:  host,
			Port:  port,
			State: StateOpen,
		}
	}

	for i := 0; i < jobs; i++ {
		if err := p.submit(
			context.Background(),
			"localhost",
			uint16(i+1),
			fn,
		); err != nil {
			t.Fatal(err)
		}
	}

	go func() {
		for range p.readResults() {
		}
	}()

	if err := p.shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	if maxRunning.Load() > concurrency {
		t.Fatalf(
			"max concurrency %d exceeded: %d",
			concurrency,
			maxRunning.Load(),
		)
	}
}

func TestPool_SubmitAndResult(t *testing.T) {
	p := newPool(2)

	err := p.submit(
		context.Background(),
		"localhost",
		80,
		func(ctx context.Context, host string, port uint16) Result {
			return Result{
				Host:  host,
				Port:  port,
				State: StateOpen,
			}
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	result, ok := <-p.readResults()
	if !ok {
		t.Fatal("expected result, got closed channel")
	}

	if result.Host != "localhost" {
		t.Fatalf("expected localhost, got %s", result.Host)
	}

	if result.Port != 80 {
		t.Fatalf("expected port 80, got %d", result.Port)
	}

	if result.State != StateOpen {
		t.Fatalf("expected open, got %s", result.State)
	}

	_, ok = <-p.readResults()
	if ok {
		t.Fatal("expected results channel to be closed")
	}
}

func TestPool_MultipleJobs(t *testing.T) {
	const jobs = 100

	p := newPool(4)

	var wg sync.WaitGroup
	wg.Add(1)

	var count atomic.Int32

	go func() {
		defer wg.Done()

		for range p.readResults() {
			count.Add(1)
		}
	}()

	for i := 0; i < jobs; i++ {
		err := p.submit(
			context.Background(),
			"localhost",
			uint16(i+1),
			func(ctx context.Context, host string, port uint16) Result {
				return Result{
					Host:  host,
					Port:  port,
					State: StateOpen,
				}
			},
		)

		if err != nil {
			t.Fatalf("submit job %d: %v", i, err)
		}
	}

	if err := p.shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if got := count.Load(); got != jobs {
		t.Fatalf("expected %d results, got %d", jobs, got)
	}
}

func TestPool_SlowResultReader(t *testing.T) {
	const jobs = 100

	p := newPool(4)

	var readerWG sync.WaitGroup
	readerWG.Add(1)

	var received atomic.Int32

	go func() {
		defer readerWG.Done()

		for range p.readResults() {
			received.Add(1)

			time.Sleep(time.Millisecond)
		}
	}()

	submitDone := make(chan error, 1)

	go func() {
		for i := 0; i < jobs; i++ {
			err := p.submit(
				context.Background(),
				"localhost",
				uint16(i+1),
				func(ctx context.Context, host string, port uint16) Result {
					return Result{
						Host:  host,
						Port:  port,
						State: StateOpen,
					}
				},
			)

			if err != nil {
				submitDone <- err
				return
			}
		}

		submitDone <- nil
	}()

	select {
	case err := <-submitDone:
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Submit blocked for too long")
	}

	if err := p.shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	readerWG.Wait()

	if got := received.Load(); got != jobs {
		t.Fatalf(
			"expected %d results, got %d",
			jobs,
			got,
		)
	}
}
