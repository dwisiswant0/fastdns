package fastdns

import (
	"time"

	"github.com/slavc/xdp"
)

// FastDNSOption is a functional option for configuring FastDNS.
type FastDNSOption func(*FastDNS)

// WithSrcLink sets the source link for FastDNS.
// This is the link that will be used to send DNS queries.
// It is typically the link that is used to reach the DNS resolver.
// The link should be a valid Link object with a valid IP and MAC address.
// If not set, the default link will be used.
// The default link is determined by the system's routing table.
func WithSrcLink(link Link) FastDNSOption {
	return func(f *FastDNS) {
		f.src = link
	}
}

// WithDstLink sets the destination link for FastDNS.
// This is the link that will be used to receive DNS responses.
// It is typically the link that is used to receive DNS responses.
// The link should be a valid Link object with a valid IP and MAC address.
// If not set, the default link will be used.
// The default link is determined by the system's routing table.
func WithDstLink(link Link) FastDNSOption {
	return func(f *FastDNS) {
		f.dst = link
	}
}

// WithNIC sets the network interface card (NIC) for FastDNS.
// This is the NIC that will be used to send and receive DNS packets.
// The NIC should be a valid string representing the name of the network
// interface.
// If not set, the default NIC will be used.
// The default NIC is determined by the system's routing table.
func WithNIC(nic string) FastDNSOption {
	return func(f *FastDNS) {
		f.nic = nic
	}
}

// WithTimeout sets the timeout for DNS queries.
// This is the maximum time that FastDNS will wait for a DNS response.
// The timeout should be a valid time.Duration object.
// If not set, the default timeout will be used.
// The default timeout is defined in the [DefaultTimeout] constant.
func WithTimeout(timeout time.Duration) FastDNSOption {
	return func(f *FastDNS) {
		f.timeout = timeout
	}
}

// WithQueueID sets the queue ID for FastDNS.
// This is the queue that will be used to send DNS packets.
// The queue ID should be a valid integer.
// If not set, the default queue ID (0) will be used.
func WithQueueID(queueID int) FastDNSOption {
	return func(f *FastDNS) {
		f.queueID = queueID
	}
}

// WithProgram sets the XDP program for FastDNS.
// This is the program that will be used to process DNS packets.
// The program should be a valid [xdp.Program] object.
// If not set, the default program will be used.
func WithProgram(program *xdp.Program) FastDNSOption {
	return func(f *FastDNS) {
		f.program = program
	}
}

// WithSocket sets the XDP socket for FastDNS.
// This is the socket that will be used to send and receive DNS packets.
// The socket should be a valid [xdp.Socket] object.
// If not set, the default socket will be used.
func WithSocket(socket *xdp.Socket) FastDNSOption {
	return func(f *FastDNS) {
		f.socket = socket
	}
}

// WithSocketOptions sets the XDP socket options for FastDNS.
// These options control the behavior of the socket.
// The options should be a valid [xdp.SocketOptions] object.
// If not set, the default options will be used.
// The default options are defined in the [xdp.DefaultSocketOptions] variable.
func WithSocketOptions(options *xdp.SocketOptions) FastDNSOption {
	return func(f *FastDNS) {
		f.socketOpts = options
	}
}

// WithCacheTTL sets the time-to-live (TTL) for cached DNS responses.
// This is the maximum time that FastDNS will keep a DNS response in the cache.
// The TTL should be a valid time.Duration object.
// If not set, the default TTL will be used.
// The default TTL is defined in the [DefaultCacheTTL] constant.
func WithCacheTTL(ttl time.Duration) FastDNSOption {
	return func(f *FastDNS) {
		f.cacheTTL = ttl
	}
}

// WithCacheCapacity sets the capacity for the DNS response cache.
// This is the maximum number of DNS responses that FastDNS will keep in the
// cache.
// The capacity should be a valid integer.
// If not set, the default capacity will be used.
// The default capacity is defined in the [DefaultCacheCapacity] constant.
func WithCacheCapacity(capacity int) FastDNSOption {
	return func(f *FastDNS) {
		f.cacheCapacity = capacity
	}
}

// WithNoCache disables the DNS response cache.
// If set, FastDNS will not cache any DNS responses.
// This is useful for testing or debugging purposes.
// If not set, the cache will be enabled by default.
// The default behavior is to cache DNS responses.
func WithNoCache() FastDNSOption {
	return func(f *FastDNS) {
		f.cacheDisabled = true
	}
}

// WithMaxRetries sets the maximum number of retries for DNS queries.
// This is the maximum number of times that FastDNS will retry a DNS query
// if it fails.
// The number of retries should be a valid integer.
// If not set, the default number of retries will be used.
// The default number of retries is defined in the [DefaultMaxRetries]
// constant.
func WithMaxRetries(retries int) FastDNSOption {
	return func(f *FastDNS) {
		f.maxRetries = retries
	}
}

// WithAllowNonGlobalIPs allows FastDNS to use non-global-unicast IPv4 addresses
// (such as loopback, multicast, or link-local) for source and destination IPs.
// By default, only global unicast addresses are allowed.
func WithAllowNonGlobalIPs() FastDNSOption {
	return func(f *FastDNS) {
		f.allowNonGlobalIPs = true
	}
}

// WithMaxBatchSize sets the maximum batch size for DNS queries.
// This is the maximum number of DNS queries that FastDNS will send in a single
// batch.
// If not set, the batch size will be determined dynamically based on the number
// of free TX slots available.
func WithMaxBatchSize(size int) FastDNSOption {
	return func(f *FastDNS) {
		f.maxBatchSize = size
	}
}
