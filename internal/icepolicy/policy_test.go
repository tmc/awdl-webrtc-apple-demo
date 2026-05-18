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

func TestPublishCandidateRawHostCandidates(t *testing.T) {
	policy := Policy{RawHostCandidates: true}
	candidate := "candidate:1 1 udp 2130706431 fd00::1 12345 typ host ufrag test"
	got := policy.PublishCandidate(candidate, net.ParseIP("fe80::1"))
	if !strings.Contains(got, " fe80::1 12345 typ host ") {
		t.Fatalf("host candidate was not rewritten: %s", got)
	}
}

func TestPublishCandidateIgnoresNonSyntheticCandidates(t *testing.T) {
	policy := Policy{RawHostCandidates: true}
	candidate := "candidate:1 1 udp 2130706431 10.0.0.1 12345 typ host ufrag test"
	if got := policy.PublishCandidate(candidate, net.ParseIP("10.0.0.1")); got != candidate {
		t.Fatalf("candidate changed unexpectedly: %s", got)
	}
}

func TestPublishDisabled(t *testing.T) {
	sdp := "a=candidate:1 1 udp 2130706431 fd00::1 12345 typ host ufrag test"
	if got := (Policy{}).Publish(sdp, net.ParseIP("fe80::1")); got != sdp {
		t.Fatalf("Publish changed disabled policy SDP: %q", got)
	}
}

func TestStripSDPCandidates(t *testing.T) {
	sdp := strings.Join([]string{
		"v=0",
		"a=mid:0",
		"a=candidate:1 1 udp 2130706431 fd00::1 12345 typ host ufrag test",
		"a=end-of-candidates",
		"a=setup:actpass",
	}, "\n")
	got := StripSDPCandidates(sdp)
	if strings.Contains(got, "a=candidate:") || strings.Contains(got, "a=end-of-candidates") {
		t.Fatalf("candidate attributes were not stripped:\n%s", got)
	}
	if !strings.Contains(got, "a=mid:0") || !strings.Contains(got, "a=setup:actpass") {
		t.Fatalf("non-candidate attributes were stripped:\n%s", got)
	}
}
