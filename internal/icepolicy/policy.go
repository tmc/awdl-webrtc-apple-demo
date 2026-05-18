package icepolicy

import (
	"net"
	"strings"

	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

// Policy controls how the demo publishes host candidates.
type Policy struct {
	RawHostCandidates bool
}

// Configure applies candidate publication settings to the Pion setting engine.
func (p Policy) Configure(se *webrtc.SettingEngine, mdnsMode ice.MulticastDNSMode, localIP net.IP) {
	if mdnsMode != ice.MulticastDNSModeDisabled || !p.UsesSyntheticHostCandidate(localIP) {
		return
	}
	se.SetNAT1To1IPs([]string{syntheticHostCandidateIP(localIP) + "/" + localIP.String()}, webrtc.ICECandidateTypeHost)
}

// UsesSyntheticHostCandidate reports whether the policy needs Pion to gather a
// synthetic host candidate that is published as the selected link-local IP.
func (p Policy) UsesSyntheticHostCandidate(localIP net.IP) bool {
	return p.RawHostCandidates && localIP != nil && localIP.To4() == nil && localIP.IsLinkLocalUnicast()
}

// Publish rewrites gathered host candidates to the selected local IP when the
// raw host-candidate policy is enabled.
//
// Deprecated: use PublishCandidate with explicit candidate signaling instead.
func (p Policy) Publish(sdp string, localIP net.IP) string {
	if !p.UsesSyntheticHostCandidate(localIP) {
		return sdp
	}
	return rewriteHostCandidateAddress(sdp, localIP)
}

// PublishCandidate rewrites one gathered host candidate to the selected local
// IP when the raw host-candidate policy is enabled.
func (p Policy) PublishCandidate(candidate string, localIP net.IP) string {
	if !p.UsesSyntheticHostCandidate(localIP) {
		return candidate
	}
	return rewriteHostCandidateAddressLine(candidate, localIP)
}

// StripSDPCandidates removes trickled candidate attributes from an SDP blob.
func StripSDPCandidates(sdp string) string {
	lines := strings.Split(sdp, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if strings.HasPrefix(trimmed, "a=candidate:") || trimmed == "a=end-of-candidates" {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func syntheticHostCandidateIP(ip net.IP) string {
	if ip.To4() != nil {
		return "198.18.0.1"
	}
	return "fd00::1"
}

func rewriteHostCandidateAddress(sdp string, ip net.IP) string {
	if ip == nil {
		return sdp
	}
	lines := strings.Split(sdp, "\n")
	for i, line := range lines {
		suffix := ""
		if strings.HasSuffix(line, "\r") {
			suffix = "\r"
		}
		lines[i] = rewriteHostCandidateAddressLine(strings.TrimRight(line, "\r"), ip) + suffix
	}
	return strings.Join(lines, "\n")
}

func rewriteHostCandidateAddressLine(line string, ip net.IP) string {
	if ip == nil {
		return line
	}
	prefix := ""
	candidate := line
	if strings.HasPrefix(candidate, "a=") {
		prefix = "a="
		candidate = strings.TrimPrefix(candidate, "a=")
	}
	if !strings.HasPrefix(candidate, "candidate:") {
		return line
	}
	fields := strings.Fields(candidate)
	if len(fields) < 8 || fields[7] != "host" {
		return line
	}
	fields[4] = ip.String()
	return prefix + strings.Join(fields, " ")
}
