package netx

import (
	"context"
	"net"
	"time"
)

// ContextedDialer is an interface that wraps the basic DialContext method.
type ContextedDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Dialer is an interface that wraps the basic dial methods that has the same
// signature as in standard library.
type Dialer interface {
	ContextedDialer
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
	Dial(network, address string) (net.Conn, error)
}

// WrapContexted wraps a ContextedDialer to a Dialer by adding other methods on
// it. The Dial method of the returned Dialer will have a default timeout,
// which is 1 minute.
func WrapContexted(cd ContextedDialer) Dialer {
	return wrapper{cd}
}

type wrapper struct {
	ContextedDialer
}

func (w wrapper) Dial(network, address string) (net.Conn, error) {
	return w.DialTimeout(network, address, defaultDialTimeout)
}

func (w wrapper) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return w.DialContext(ctx, network, address)
}
