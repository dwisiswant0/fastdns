package fastdns

import "net"

// Link represents a network link with its IP and MAC address.
type Link struct {
	// IP is the IP address of the link.
	IP net.IP

	// MAC is the MAC address of the link.
	MAC net.HardwareAddr
}
