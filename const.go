package fastdns

import "time"

const (
	// DefaultTimeout is the default timeout for DNS queries.
	DefaultTimeout = 5 * time.Second

	// DefaultMaxRetries is the default maximum number of retries for DNS
	// queries.
	DefaultMaxRetries = 3

	// DefaultQueueID is the default queue ID for sending DNS packets.
	DefaultQueueID = 0

	// DefaultCacheTTL is the default time-to-live for cached DNS responses.
	DefaultCacheTTL = 1 * time.Minute

	// DefaultCacheCapacity is the default capacity for the DNS response cache.
	DefaultCacheCapacity = 1000
)

const (
	defaultMaxBackoff = 10_000 // cap at 10μs
)
