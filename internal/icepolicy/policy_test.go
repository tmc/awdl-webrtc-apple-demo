package icepolicy

import (
	"net"
	"strings"
	"testing"
)

func TestPublishRawHostCandidates(t *testing.T) {
	policy := Policy{RawHostCandidates: true}
	sdp := strings.Join([]string{
		"v=0",
		"a=candidate:1 1 udp 2130706431 fd00::1 12345 typ host ufrag test",
		"a=candidate:2 1 udp 1694498815 192.0.2.1 54321 typ srflx raddr 0.0.0.0 rport 0",
	}, "\n")
	got := policy.Publish(sdp, net.ParseIP("fe80::1"))
	if !strings.Contains(got, " fe80::1 12345 typ host ") {
		t.Fatalf("host candidate was not rewritten:\n%s", got)
	}
	if !strings.Contains(got, " 192.0.2.1 54321 typ srflx ") {
		t.Fatalf("srflx candidate changed unexpectedly:\n%s", got)
	}
}

func TestPublishDisabled(t *testing.T) {
	sdp := "a=candidate:1 1 udp 2130706431 fd00::1 12345 typ host ufrag test"
	if got := (Policy{}).Publish(sdp, net.ParseIP("fe80::1")); got != sdp {
		t.Fatalf("Publish changed disabled policy SDP: %q", got)
	}
}
