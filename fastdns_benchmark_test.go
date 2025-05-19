package fastdns_test

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/dwisiswant0/fastdns"
	"github.com/miekg/dns"
)

func BenchmarkQuery(b *testing.B) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("cloudflare.com"), dns.TypeA)

	f, err := fastdns.New(net.ParseIP(resolver), fastdns.WithNoCache())
	if err != nil {
		b.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	pktSent := 0

	b.ResetTimer()
	for b.Loop() {
		resp := f.Query(msg)
		if resp.IsSuccess() {
			pktSent++
		} else {
			b.Logf("Failed to send DNS query: %v", err)
		}
	}

	if err := f.Close(); err != nil {
		b.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	if pktSent > 0 {
		b.Logf("Sent %d/%d packets in %v", pktSent, b.N, b.Elapsed())
	}
}

func BenchmarkQueries(b *testing.B) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("cloudflare.com"), dns.TypeA)

	f, err := fastdns.New(net.ParseIP(resolver), fastdns.WithNoCache())
	if err != nil {
		b.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	pktSent := 0

	b.ResetTimer()
	for b.Loop() {
		resp := f.Query(msg)
		if resp.IsSuccess() {
			pktSent++
		} else {
			b.Logf("Failed to send DNS query: %v", err)
		}
	}

	if err := f.Close(); err != nil {
		b.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	if pktSent > 0 {
		b.Logf("Sent %d/%d packets in %v", pktSent, b.N, b.Elapsed())
	}
}

func BenchmarkCompare(b *testing.B) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("cloudflare.com"), dns.TypeA)

	f, err := fastdns.New(net.ParseIP(resolver), fastdns.WithNoCache())
	if err != nil {
		b.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during cleanup: %v\n", err)
		}
	}()

	client := &dns.Client{
		Net: "udp",
	}

	b.Run("miekg", func(b *testing.B) {
		pktSent := 0

		b.ResetTimer()
		for b.Loop() {
			_, _, err = client.Exchange(msg, resolver+":53")
			if err == nil {
				pktSent++
			} else {
				b.Logf("Failed to send DNS query: %v", err)
			}
		}

		if pktSent > 0 {
			b.Logf("Sent %d/%d packets in %v", pktSent, b.N, b.Elapsed())
		}
	})

	b.Run("fastdns", func(b *testing.B) {
		pktSent := 0

		b.ResetTimer()
		for b.Loop() {
			resp := f.Query(msg)
			if resp.IsSuccess() {
				pktSent++
			} else {
				b.Logf("Failed to send DNS query: %v", err)
			}
		}

		if pktSent > 0 {
			b.Logf("Sent %d/%d packets in %v", pktSent, b.N, b.Elapsed())
		}
	})
}
