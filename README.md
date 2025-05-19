# FastDNS

[![GoDoc](https://pkg.go.dev/static/frontend/badge/badge.svg)](http://pkg.go.dev/github.com/dwisiswant0/fastdns)

**fastdns** is a high-performance DNS client for Go. It leverages Linux XDP sockets for ultra-low-latency, lock-free DNS queries, bypassing the kernel network stack for maximum throughput.

## Install

```bash
go get github.com/dwisiswant0/fastdns@latest
```

## Example

```go
package main

import (
    "fmt"
    "net"

    "github.com/dwisiswant0/fastdns"
    "github.com/miekg/dns"
)

func main() {
	// Define the DNS resolver to use.
	// This should be a valid IP address of a DNS resolver.
	resolver := net.ParseIP("1.1.1.1")

	// Create a new FastDNS instance with the default resolver.
	f, err := fastdns.New(resolver)
	if err != nil {
		panic(err)
	}

	// Ensure that the FastDNS instance is closed when done.
	// Closing is essential to properly release all underlying XDP and socket
	// resources.
	//
	// If Close is not called, system resources such as file descriptors and
	// network handles may be leaked, which can eventually exhaust available
	// resources and cause failures in network operations or prevent new FastDNS
	// instances from being created.
	defer f.Close()

	// Create a new DNS query message.
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("cloudflare.com"), dns.TypeA)

	// Send the DNS query and receive the response.
	resp, _, err := f.Query(msg)
	if err != nil {
		f.Close()

		panic(err)
	}

	fmt.Printf("Response: %v\n", resp.String())
	fmt.Printf("Round-trip time: %v\n", rtt)
}
```

## Status

> [!CAUTION]
> **fastdns** has NOT reached 1.0 yet. Therefore, this library is currently not supported and does not offer a stable API; use at your own risk.

There are no guarantees of stability for the APIs in this library, and while they are not expected to change dramatically. API tweaks and bug fixes may occur.

## License

**fastdns** is released by **[@dwisiswant0](https://github.com/dwisiswant0)** under the Apache 2.0 license. See [LICENSE](/LICENSE).