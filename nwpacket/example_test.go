//go:build darwin

package nwpacket_test

import (
	"fmt"
	"net"

	applenetwork "github.com/tmc/apple/network"
	"github.com/tmc/awdl-webrtc-apple-demo/nwpacket"
)

func ExampleConfig() {
	cfg := nwpacket.Config{
		InterfaceName:         "awdl0",
		LocalAddr:             &net.UDPAddr{IP: net.ParseIP("fe80::1"), Zone: "awdl0"},
		RequiredInterfaceType: applenetwork.NWInterfaceTypeWifi,
		SetRequiredInterface:  true,
		IncludePeerToPeer:     true,
		RequireInterface:      true,
		ReuseLocalAddress:     true,
	}
	fmt.Println(cfg.InterfaceName, cfg.IncludePeerToPeer)
	// Output: awdl0 true
}
