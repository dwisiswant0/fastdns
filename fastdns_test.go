package fastdns_test

import (
	"net"
	"os"
	"testing"

	"github.com/dwisiswant0/fastdns"
	"github.com/miekg/dns"
)

var resolver = "1.1.1.1"

func TestQuery(t *testing.T) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("cloudflare.com"), dns.TypeA)

	f, err := fastdns.New(net.ParseIP(resolver))
	if err != nil {
		t.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	resp := f.Query(msg)
	if err := resp.Err(); err != nil {
		_ = f.Close()

		t.Fatalf("Failed to send DNS query: %v", err)
	}

	if !resp.IsSuccess() {
		_ = f.Close()

		t.Fatal("Query was not successful")
	}

	if err := f.Close(); err != nil {
		t.Fatalf("Failed to initialize FastDNS: %v", err)
	}
}

func TestQueries(t *testing.T) {
	fqdns := []string{
		"cloudflare.com",
		"google.com",
	}

	msgs := make([]*dns.Msg, len(fqdns))
	for i, fqdn := range fqdns {
		msgs[i] = new(dns.Msg)
		msgs[i].SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
	}

	f, err := fastdns.New(net.ParseIP(resolver))
	if err != nil {
		t.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("Failed to close FastDNS: %v", err)
		}
	}()

	resps, err := f.Queries(msgs...)
	if len(resps) != len(msgs) {
		t.Fatalf("Expected %d responses, got %d", len(msgs), len(resps))
	}

	if err != nil {
		t.Fatalf("Failed to send DNS queries: %v", err)
	}

	for i, resp := range resps {
		if err := resp.Err(); err != nil {
			t.Errorf("Failed to send DNS query for %s: %v", fqdns[i], err)
			continue
		}

		if !resp.IsSuccess() {
			t.Errorf("Query for %s was not successful", fqdns[i])
		}

		if resp.IsSuccess() {
			t.Logf("Query for %s was successful", fqdns[i])
			t.Logf("Response: %s", resp.String())
			t.Logf("Round-trip time: %v", resp.RTT)
		}
	}
}

func TestQueryFromFile(t *testing.T) {
	f, err := fastdns.New(net.ParseIP(resolver))
	if err != nil {
		t.Fatalf("Failed to initialize FastDNS: %v", err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("Failed to close FastDNS: %v", err)
		}
	}()

	file, err := os.Open("testdata/domains-2.txt")
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}

	resps, err := f.QueryFromFile(file, func(s string) *dns.Msg {
		msg := new(dns.Msg)
		msg.SetQuestion(dns.Fqdn(s), dns.TypeA)
		msg.RecursionDesired = true

		return msg
	})

	if err != nil {
		t.Fatalf("Failed to query from file: %v", err)
	}

	for _, resp := range resps {
		if err := resp.Err(); err != nil {
			t.Errorf("Failed to send DNS query: %v", err)
			continue
		}

		question := resp.Message.Question[0]

		if !resp.IsSuccess() {
			t.Errorf("Query for %s was not successful", question.String())
		}

		if resp.IsSuccess() {
			t.Logf("Query for %s was successful", question.String())
			t.Logf("Response: %s", resp.String())
			t.Logf("Round-trip time: %v", resp.RTT)
		}
	}
}
