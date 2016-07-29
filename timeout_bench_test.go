package netx

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

type timeouterror struct {
}

func (t *timeouterror) Error() string {
	return ioTimeout
}

func (t *timeouterror) Timeout() bool {
	return true
}

func (t *timeouterror) Temporary() bool {
	return false
}

// This is the slowest
func BenchmarkTimeoutUsingCast(b *testing.B) {
	var err error = &timeouterror{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err.(net.Error).Timeout() {
		}
	}
}

// This is surprisingly slow
func BenchmarkTimeoutUsingBytesEquals(b *testing.B) {
	var err error = &timeouterror{}
	ioTimeoutBytes := []byte(ioTimeout)
	iotl := len(ioTimeoutBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es := []byte(err.Error())
		esl := len(es)
		if esl >= iotl && bytes.Equal(es[esl-ioTimeoutLength:], ioTimeoutBytes) {
		}
	}
}

// This is also surprisingly slow
func BenchmarkTimeoutUsingHandBuiltCompare(b *testing.B) {
	var err error = &timeouterror{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es := err.Error()
		if hasSuffix(es, ioTimeout, ioTimeoutLength) {
		}
	}
}

// This is faster
func BenchmarkTimeoutUsingSuffix(b *testing.B) {
	var err error = &timeouterror{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es := err.Error()
		if strings.HasSuffix(es, ioTimeout) {
		}
	}
}

// This is fastest
func BenchmarkTimeoutUsingSliceCompare(b *testing.B) {
	var err error = &timeouterror{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		es := err.Error()
		esl := len(es)
		if esl >= ioTimeoutLength && es[esl-ioTimeoutLength:] == ioTimeout {
		}
	}
}

func hasSuffix(a string, b string, l int) bool {
	delta := len(a) - l
	if delta < 0 {
		return false
	}
	for i := 0; i < l; i++ {
		if a[i+delta] != b[i] {
			return false
		}
	}
	return true
}
