package tcpscanner

import (
	"context"
	"net"
	"sync"
	"time"
)

type pool struct {
	workers int
	jobs    chan job
	results chan Result
	//errs    chan error

	wg sync.WaitGroup
}

type job struct {
	host string
	ip   net.IP
	port uint16
	ctx  context.Context
	fn   func(ctx context.Context) error
}

func newPool(size int, timeout time.Duration) *pool {
	p := &pool{
		workers: size,
		jobs:    make(chan job, maxPort),
		//errs:    make(chan error, maxPort),
		results: make(chan Result, maxPort),
	}

	return p
}

//test
