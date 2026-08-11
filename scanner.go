package tcpscanner

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

type Scanner struct {
	concurrency int
	timeout     time.Duration
}

func New(opts ...Option) (*Scanner, error) {
	s := &Scanner{concurrency: 100, timeout: 500 * time.Millisecond}

	for _, opt := range opts {
		opt(s)
	}

	if s.concurrency <= 0 {
		return nil, errors.New("Concurrency must be >0")
	}

	if s.timeout <= 0 {
		return nil, errors.New("Timeout must be >0")
	}

	return s, nil
}

func (s *Scanner) Scan(ctx context.Context, hosts []string, ran []uint16) ([]Result, error) {
	res := make([]Result, 0, len(hosts)*len(ran))
	pool := newPool(s.concurrency)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for r := range pool.readResults() {
			res = append(res, r)
		}
	}()

	shutdown := func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pool.shutdown(shutCtx)
		wg.Wait()
	}

	var checkPort = func(ctx context.Context, host string, port uint16) Result {
		start := time.Now()
		resSingle := Result{
			Host: host, Port: port,
		}

		if ip := net.ParseIP(host); ip != nil {
			resSingle.IP = ip
		}

		dialer := net.Dialer{
			Timeout: s.timeout,
		}

		conn, err := dialer.DialContext(
			ctx,
			"tcp",
			net.JoinHostPort(host, strconv.Itoa(int(port))),
		)

		resSingle.Duration = time.Since(start)

		if err != nil {
			resSingle.State = classifyError(err)
			resSingle.Err = err
			return resSingle
		}

		defer conn.Close()

		if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
			resSingle.IP = tcpAddr.IP
		}

		resSingle.State = StateOpen
		return resSingle
	}

	for _, i := range hosts {
		for _, p := range ran {
			if err := pool.submit(ctx, i, p, checkPort); err != nil {
				shutdown()
				return res, err
			}
		}
	}

	shutdown()
	return res, nil
}
