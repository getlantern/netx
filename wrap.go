package netx

import (
	"net"
)

// WrappedConn is a connection that wraps another connection.
type WrappedConn interface {
	net.Conn
	Wrapped() net.Conn
}

// WalkWrapped walks the tree of wrapped conns, calling the callback. If
// callback returns false, the walk stops.
func WalkWrapped(conn net.Conn, cb func(net.Conn) bool) {
	for {
		switch t := conn.(type) {
		case WrappedConn:
			if !cb(t) {
				return
			}
			conn = t.Wrapped()
		default:
			return
		}
	}
}
