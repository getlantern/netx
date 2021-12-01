package netx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveLocalhost ensures that localhost resolves to its IPv4 address when the network
// parameter is ambiguous. This can be important as local servers are unlikely to listen over IPv6
// except in special cases.
func TestResolveLocalhost(t *testing.T) {
	// Try several times as results are randomized.
	for i := 0; i < 10; i++ {
		addr, err := Resolve("tcp", "localhost:999")
		require.NoError(t, err)
		require.NotNil(t, addr.IP.To4(), "IP (%v) seems to be IPv6, but should be IPv4", addr.IP)

		udpAddr, err := ResolveUDPAddr("udp", "localhost:999")
		require.NoError(t, err)
		require.NotNil(t, udpAddr.IP.To4(), "IP (%v) seems to be IPv6, but should be IPv4", udpAddr.IP)

		// Specifying IPv6 should get an IPv6 result.
		addr, err = Resolve("tcp6", "localhost:999")
		require.NoError(t, err)
		require.Nil(t, addr.IP.To4(), "IP (%v) seems to be IPv4, but should be IPv6", addr.IP)

		udpAddr, err = ResolveUDPAddr("udp6", "localhost:999")
		require.NoError(t, err)
		require.Nil(t, udpAddr.IP.To4(), "IP (%v) seems to be IPv4, but should be IPv6", udpAddr.IP)
	}
}
