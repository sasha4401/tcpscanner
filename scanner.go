package tcpscanner

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"
)

type Scanner struct {
	Concurrency int
	Timeout     time.Duration
}

func New(opts ...Option) (*Scanner, error) {
	s := &Scanner{Concurrency: 100, Timeout: 500 * time.Microsecond}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *Scanner) Scan(ctx context.Context, hosts []string, ran []uint16) ([]Result, error) {
	res := make([]Result, 0, len(hosts)*len(ran))
	pool := newPool(s.Concurrency)
	go func() {
		for {
			select {
			case r := <-pool.Results():
				res = append(res, r)
			case <-ctx.Done():
				ctxShutdown, cancelShut := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelShut()

				if err := pool.Shutdown(ctxShutdown); err != nil {
					//шатдаун не успел завершиться
				}

				return
			}
		}
	}()

	var checkPort = func(ctx context.Context, host string, port uint16) Result {
		start := time.Now()
		resSingle := Result{
			Host: host, Port: port,
		}

		dialer := net.Dialer{
			Timeout: s.Timeout,
		}

		conn, err := dialer.DialContext(
			ctx,
			"tcp",
			net.JoinHostPort(host, strconv.Itoa(int(port))),
		)

		resSingle.Duration = time.Since(start)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				resSingle.State = StateCanceled
			} else if errors.Is(err, context.DeadlineExceeded) {
				resSingle.State = StateTimeout
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				resSingle.State = StateTimeout
			} else {
				resSingle.State = StateClosed
			}

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
			ctxJob, cancel := context.WithTimeout(ctx, s.Timeout)
			if err := pool.Submit(ctxJob, i, p, checkPort, cancel); err != nil {
				cancel()
				return nil, err
			}
		}
	}

	if err := pool.Shutdown(ctx); err != nil {
		return nil, err
	}

	return res, nil
}
