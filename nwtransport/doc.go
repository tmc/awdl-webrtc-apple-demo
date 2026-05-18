//go:build darwin

// Package nwtransport adapts Network.framework UDP listeners to Pion's
// transport.Net interface.
//
// The package is intentionally small: UDP ListenPacket calls use nwpacket,
// while DNS, TCP, and connected UDP operations fall back to Pion's standard
// network implementation. This is enough to demonstrate Pion ICE gathering and
// connectivity over Apple-only link policies without an ICE UDP mux.
package nwtransport
