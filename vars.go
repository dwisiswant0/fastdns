package fastdns

import (
	"errors"
)

var (
	ErrARPEntryMissing          = errors.New("ARP entry might be missing or stale")
	ErrInvalidFile              = errors.New("could not read file")
	ErrInvalidIPAddr            = errors.New("invalid IP address")
	ErrInvalidMACAddr           = errors.New("invalid MAC address")
	ErrInvalidPacketDestination = errors.New("packet not destined for us or not from DNS server")
	ErrInvalidQueueID           = errors.New("invalid queue ID")
	ErrInvalidTimeout           = errors.New("invalid timeout")
	ErrInvalidXDPSocket         = errors.New("missing or invalid XDP socket")
	ErrNoDefaultRoute           = errors.New("could not determine default route")
	ErrNoDestMACAddr            = errors.New("could not find a valid/reachable MAC address")
	ErrNoDNSResponse            = errors.New("no DNS response")
	ErrNoDstIP                  = errors.New("could not determine destination IP")
	ErrNoFile                   = errors.New("no file provided")
	ErrNoIPLayer                = errors.New("could not get IP layer")
	ErrNoIPv4Addr               = errors.New("could not get IPv4 address")
	ErrNoLink                   = errors.New("could not get link")
	ErrNoMACAddr                = errors.New("could not get MAC address")
	ErrNoMapperFunc             = errors.New("no mapper function provided")
	ErrNoNeighbors              = errors.New("could not list neighbors")
	ErrNoNextHopIP              = errors.New("could not determine next hop IP")
	ErrNoQueries                = errors.New("no queries provided")
	ErrNoSuitableIPv4Addr       = errors.New("no suitable IPv4 address found")
	ErrNoTXDescriptors          = errors.New("could not get TX descriptors")
	ErrNoUDPLayer               = errors.New("could not get UDP layer")
	ErrPacketTooSmall           = errors.New("packet too small")
	ErrPoll                     = errors.New("poll error")
)
