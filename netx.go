// Package netx provides additional libraries that extend some of the behaviors
// in the net standard package.
package netx

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/getlantern/golog"
)

var (
	log = golog.LoggerFor("netx")
)

var (
	dial               atomic.Value
	dialUDP            atomic.Value
	listenUDP          atomic.Value
	resolveTCPAddr     atomic.Value
	resolveUDPAddr     atomic.Value
	nat64Prefix        atomic.Value
	defaultDialTimeout = 1 * time.Minute
)

func init() {
	log.Debug("initializing netx")
	go keepNAT64PrefixFresh()
	Reset()
}

func keepNAT64PrefixFresh() {
	for {
		updateNAT64Prefix()
		time.Sleep(1 * time.Second)
	}
}

func updateNAT64Prefix() {
	ips, err := net.LookupIP("ipv4only.arpa")
	if err == nil {
		log.Debugf("ipv4only.arpa returned %v", ips)
		for _, ip := range ips {
			if ip.To4() == nil {
				prefix := ip[:12]
				log.Debugf("Got nat64 prefix: %v", prefix)
				nat64Prefix.Store([]byte(prefix))
				return
			}
		}
	}
	log.Debug("No nat64 prefix")
	nat64Prefix.Store(nil)
}

// getNAT64Prefix returns previously fetched ipv6 prefix, or gets a fresh one using DNS lookup
func getNAT64Prefix() []byte {
	if prefix, ok := nat64Prefix.Load().([]byte); ok {
		return prefix
	}
	return nil
}

// convertAddressDNS64 takes the IP address, converts it to ipv6 and applies DNS64 prefix
func convertAddressDNS64(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	ip := net.ParseIP(host)
	if ip.To4() == nil { // if it's ipv6 already - don't do anything
		return addr
	}
	prefix := getNAT64Prefix()
	if prefix == nil {
		return addr
	}
	ipv6 := ip.To16()
	copy(ipv6[:12], prefix)
	return net.JoinHostPort(ipv6.String(), port)
}

// Dial is like DialTimeout using a default timeout of 1 minute.
func Dial(network string, addr string) (net.Conn, error) {
	log.Debugf("dial (%v) %v", network, addr)
	return DialTimeout(network, addr, defaultDialTimeout)
}

// DialUDP acts like Dial but for UDP networks.
func DialUDP(network string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error) {
	log.Debugf("dialUDP (%v) %v", network, raddr)
	return dialUDP.Load().(func(string, *net.UDPAddr, *net.UDPAddr) (*net.UDPConn, error))(network, laddr, raddr)
}

// DialTimeout dials the given addr on the given net type using the configured
// dial function, timing out after the given timeout.
func DialTimeout(network string, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	conn, err := DialContext(ctx, network, addr)

	cancel()
	return conn, err
}

// DialContext dials the given addr on the given net type using the configured
// dial function, with the given context.
func DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	log.Debugf("dialing (%v) %v", network, addr)
	addr = convertAddressDNS64(addr)
	log.Debugf("actually dialing (%v) %v", network, addr)

	dialer := dial.Load().(func(context.Context, string, string) (net.Conn, error))

	conn, err := dialer(ctx, network, addr)
	if err != nil {
		log.Errorf("error dialing (%v) %v: %v", network, addr, err)
	} else {
		log.Debugf("successfully dialed (%v) %v", network, addr)
	}
	return conn, err
}

// ListenUDP acts like ListenPacket for UDP networks.
func ListenUDP(network string, laddr *net.UDPAddr) (*net.UDPConn, error) {
	return listenUDP.Load().(func(network string, laddr *net.UDPAddr) (*net.UDPConn, error))(network, laddr)
}

// OverrideDial overrides the global dial function.
func OverrideDial(dialFN func(ctx context.Context, net string, addr string) (net.Conn, error)) {
	dial.Store(dialFN)
}

// OverrideDialUDP overrides the global dialUDP function.
func OverrideDialUDP(dialFN func(net string, laddr, raddr *net.UDPAddr) (*net.UDPConn, error)) {
	dialUDP.Store(dialFN)
}

// OverrideListenUDP overrides the global listenUDP function.
func OverrideListenUDP(listenFN func(network string, laddr *net.UDPAddr) (*net.UDPConn, error)) {
	listenUDP.Store(listenFN)
}

// Resolve resolves the given tcp address using the configured resolve function.
func Resolve(network string, addr string) (*net.TCPAddr, error) {
	return resolveTCPAddr.Load().(func(string, string) (*net.TCPAddr, error))(network, addr)
}

func ResolveUDPAddr(network string, addr string) (*net.UDPAddr, error) {
	return resolveUDPAddr.Load().(func(string, string) (*net.UDPAddr, error))(network, addr)
}

// OverrideResolve overrides the global resolve function.
func OverrideResolve(resolveFN func(net string, addr string) (*net.TCPAddr, error)) {
	resolveTCPAddr.Store(resolveFN)
}

// OverrideResolveUDP overrides the global resolveUDP function.
func OverrideResolveUDP(resolveFN func(net string, addr string) (*net.UDPAddr, error)) {
	resolveUDPAddr.Store(resolveFN)
}

// Reset resets netx to its default settings
func Reset() {
	var d net.Dialer
	OverrideDial(d.DialContext)
	OverrideDialUDP(net.DialUDP)
	OverrideListenUDP(net.ListenUDP)
	OverrideResolve(net.ResolveTCPAddr)
	OverrideResolveUDP(net.ResolveUDPAddr)
}
