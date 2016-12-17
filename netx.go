// Package netx provides additional libraries that extend some of the behaviors
// in the net standard package.
package netx

import (
	"context"
	"net"
	"sync/atomic"
	"time"
)

var (
	dial           atomic.Value
	resolveTCPAddr atomic.Value

	defaultDialTimeout = 1 * time.Minute
)

func init() {
	Reset()
}

// Dial is like DialTimeout using a default timeout of 1 minute.
func Dial(net string, addr string) (net.Conn, error) {
	return DialTimeout(net, addr, defaultDialTimeout)
}

// DialTimeout dials the given addr on the given net type using the configured
// dial function, timing out after the given timeout.
func DialTimeout(network string, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return DialContext(ctx, network, addr)
}

// DialContext dials the given addr on the given net type using the configured
// dial function, with the given context.
func DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	return dial.Load().(func(context.Context, string, string) (net.Conn, error))(ctx, network, addr)
}

// OverrideDial overrides the global dial function.
func OverrideDial(dialFN func(ctx context.Context, net string, addr string) (net.Conn, error)) {
	dial.Store(dialFN)
}

// Resolve resolves the given tcp address using the configured resolve function.
func Resolve(network string, addr string) (*net.TCPAddr, error) {
	return resolveTCPAddr.Load().(func(string, string) (*net.TCPAddr, error))(network, addr)
}

// OverrideResolve overrides the global resolve function.
func OverrideResolve(resolveFN func(net string, addr string) (*net.TCPAddr, error)) {
	resolveTCPAddr.Store(resolveFN)
}

// Reset resets netx to its default settings
func Reset() {
	var d net.Dialer
	OverrideDial(d.DialContext)
	OverrideResolve(net.ResolveTCPAddr)
}
