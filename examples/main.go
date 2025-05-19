package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/dwisiswant0/fastdns"
	"github.com/miekg/dns"
)

var (
	Resolver   string
	DomainName string
	QueueID    int
)

func main() {
	flag.StringVar(&Resolver, "resolver", "1.1.1.1", "The DNS resolver to use.")
	flag.StringVar(&DomainName, "domain", "cloudflare.com", "Domain name to use in the DNS query.")
	flag.IntVar(&QueueID, "queue", 0, "The queue on the network interface to attach to.")
	flag.Parse()

	f, err := fastdns.New(net.ParseIP(Resolver))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing network: %v\n", err)
		os.Exit(1)
	}

	// Ensure cleanup happens even if we panic
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during cleanup: %v\n", err)
		}
	}()

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(DomainName), dns.TypeA)

	resp := f.Query(msg)
	if err := resp.Err(); err != nil {
		_ = f.Close()

		fmt.Fprintf(os.Stderr, "Error sending DNS query: %v\n", err)

		os.Exit(1)
	}

	fmt.Printf("Response: %v\n", resp.String())
	fmt.Printf("Round-trip time: %v\n", resp.RTT)

	stats, err := f.Stats()
	if err != nil {
		_ = f.Close()

		fmt.Fprintf(os.Stderr, "Error getting stats: %v\n", err)

		os.Exit(1)
	}

	fmt.Printf("Stats: %+v\n", stats)
}
