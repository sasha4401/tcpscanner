package tcpscanner

import (
	"errors"
	"syscall"
)

const (
	wsaNetUnreach  = syscall.Errno(10051)
	wsaHostUnreach = syscall.Errno(10065)
	wsaConnRefused = syscall.Errno(10061)
)

func isConnectionRefused(err error) bool {
	return errors.Is(err, wsaConnRefused)
}

func isUnreachable(err error) bool {
	return errors.Is(err, wsaNetUnreach) ||
		errors.Is(err, wsaHostUnreach)
}
