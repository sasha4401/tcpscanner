package tcpscanner

import (
	"context"
	"errors"
	"net"
)

func classifyError(err error) State {
	if errors.Is(err, context.Canceled) {
		return StateCanceled
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return StateTimeout
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return StateTimeout
	}

	if isConnectionRefused(err) {
		return StateClosed
	}

	if isUnreachable(err) {
		return StateUnreachable
	}

	return StateError
}
