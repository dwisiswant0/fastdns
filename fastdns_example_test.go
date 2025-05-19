package fastdns_test

import (
	"fmt"
	"net"
	"os"

	"github.com/dwisiswant0/fastdns"
	"github.com/miekg/dns"
)

// ExampleQuery demonstrates how to use the FastDNS library to send a DNS query.
// It creates a new FastDNS instance, sends a DNS query for the A record of
// "cloudflare.com", and prints the response.
//
// This example assumes that you have a working network connection and that
// the FastDNS library is properly installed and configured.
// It also assumes that the DNS resolver is reachable and responds to the query.
func ExampleFastDNS_Query() {
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
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during cleanup: %v\n", err)
		}
	}()

	// Create a new DNS query message.
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("cloudflare.com"), dns.TypeA)

	// Send the DNS query and receive the response.
	resp := f.Query(msg)
	if err := resp.Err(); err != nil {
		_ = f.Close()

		panic(err)
	}

	// Print the response.
	for _, answer := range resp.Message.Answer {
		println(answer.String())
	}
}
