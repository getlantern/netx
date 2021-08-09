// Package netx provides additional libraries that extend some of the behaviors
// in the net standard package.
package netx

import (
	"context"
	"net"
	"strings"
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
	NAT64Prefix        atomic.Value
	defaultDialTimeout = 1 * time.Minute
)

func init() {
	log.Debug("initializing netx")
	Reset()
}

type NAT64PrefixHolder struct {
	expiration time.Time
	prefix     []byte
}

func ResetNAT64Prefix() {
	NAT64Prefix.Store(nil)
}

// getNAT64Prefix returns previously fetched ipv6 prefix, or gets a fresh one using DNS lookup
func getNAT64Prefix() []byte {
	if holder, ok := NAT64Prefix.Load().(*NAT64PrefixHolder); holder != nil && ok {
		if time.Now().Before(holder.expiration) {
			return holder.prefix
		}
	}
	ips, err := net.LookupIP("ipv4only.arpa")
	if err == nil {
		for _, ip := range ips {
			if ip.To4() == nil {
				prefix := ip[:12]
				NAT64Prefix.Store(&NAT64PrefixHolder{prefix: prefix, expiration: time.Now().Add(time.Hour * 1)})
				return prefix
			}
		}
	}
	return nil
}

// isNetworkUnreachable checks if the error matches string representation of ENETUNREACH
func isNetworkUnreachable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unreachable")
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

	dialer := dial.Load().(func(context.Context, string, string) (net.Conn, error))

	conn, err := dialer(ctx, network, addr)
	ipv4Network := network == "udp4" || network == "tcp4"
	// if we are not dialing an explicitly ipv4 network and we got ENETUNREACH - try applying DNS64 prefix
	if !ipv4Network && isNetworkUnreachable(err) {
		nat64Addr := convertAddressDNS64(addr)
		log.Debugf("failling back to dialing (%v) %v with nat64 address ", network, addr, nat64Addr)
		conn, err = dialer(ctx, network, nat64Addr)
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
