package tcpscanner

import (
	"net"
	"time"
)

type Result struct {
	Host     string
	IP       net.IP
	Port     uint16
	State    State
	Duration time.Duration
	Err      error
}

type State uint8

const (
	StateOpen State = iota
	StateClosed
	StateTimeout
	StateUnreachable
	StateCanceled
	StateError
)
