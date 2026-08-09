package tcpscanner

import (
	"context"
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

func (*Scanner) Scan(ctx context.Context, ips []string, ran []uint16) ([]Result, error) {

}
