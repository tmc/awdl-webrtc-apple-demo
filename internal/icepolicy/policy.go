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
	if !p.RawHostCandidates || mdnsMode != ice.MulticastDNSModeDisabled {
		return
	}
	if localIP == nil || localIP.To4() != nil || !localIP.IsLinkLocalUnicast() {
		return
	}
	se.SetNAT1To1IPs([]string{syntheticHostCandidateIP(localIP) + "/" + localIP.String()}, webrtc.ICECandidateTypeHost)
}

// Publish rewrites gathered host candidates to the selected local IP when the
// raw host-candidate policy is enabled.
func (p Policy) Publish(sdp string, localIP net.IP) string {
	if !p.RawHostCandidates {
		return sdp
	}
	return rewriteHostCandidateAddress(sdp, localIP)
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
		trimmed := strings.TrimRight(line, "\r")
		if !strings.HasPrefix(trimmed, "a=candidate:") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 8 || fields[7] != "host" {
			continue
		}
		fields[4] = ip.String()
		suffix := ""
		if strings.HasSuffix(line, "\r") {
			suffix = "\r"
		}
		lines[i] = strings.Join(fields, " ") + suffix
	}
	return strings.Join(lines, "\n")
}
