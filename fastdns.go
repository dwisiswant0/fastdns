package fastdns

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/maypok86/otter"
	"github.com/miekg/dns"
	"github.com/slavc/xdp"
	"github.com/vishvananda/netlink"
)

// FastDNS holds all state and resources required for high-performance DNS
// queries using XDP sockets. It manages network interface details, link-layer
// addressing, serialization buffers, and DNS response caching.
type FastDNS struct {
	// nic is the name of the network interface card
	nic string

	// src represents the source link layer information
	src Link

	// dst represents the destination link layer information
	dst Link

	// layer contains the link layer information
	// It is used to create the Ethernet, IPv4, and UDP layers
	// for the DNS query
	layer struct {
		// Ethernet is the Ethernet layer
		Ethernet *layers.Ethernet

		// IPv4 is the IPv4 layer
		IPv4 *layers.IPv4

		// UDP is the UDP layer
		UDP *layers.UDP
	}

	timeout           time.Duration
	maxRetries        int
	allowNonGlobalIPs bool

	// XDP resources
	maxBatchSize int
	queueID      int
	program      *xdp.Program
	socket       *xdp.Socket
	socketOpts   *xdp.SocketOptions

	// Serialization resources
	serializeBuf  gopacket.SerializeBuffer
	serializeOpts gopacket.SerializeOptions

	// Cache for DNS responses
	cache         otter.Cache[*dns.Msg, *dns.Msg]
	cacheTTL      time.Duration
	cacheCapacity int
	cacheDisabled bool
}

// New creates a new [FastDNS] instance with the given destination IP address
// and optional configuration options.
// It returns an error if the destination IP address is not set or if any of the
// configuration options are invalid. It also sets up the XDP program and socket
// for sending and receiving DNS queries.
func New(dstIP net.IP, opts ...FastDNSOption) (*FastDNS, error) {
	if dstIP == nil {
		return nil, ErrNoDstIP
	}

	f := new(FastDNS)
	f.src.IP = nil
	f.src.MAC = nil
	f.dst.IP = dstIP
	f.dst.MAC = nil
	f.timeout = DefaultTimeout
	f.maxRetries = DefaultMaxRetries
	f.queueID = DefaultQueueID
	f.cacheTTL = DefaultCacheTTL
	f.cacheCapacity = DefaultCacheCapacity

	for _, opt := range opts {
		opt(f)
	}

	if f.queueID < 0 {
		return nil, ErrInvalidQueueID
	}

	if f.timeout <= 0 {
		return nil, ErrInvalidTimeout
	}

	var (
		route   netlink.Route
		srcLink netlink.Link
	)

	routes, err := netlink.RouteGet(f.dst.IP)
	if err != nil || len(routes) == 0 {
		return nil, fmt.Errorf("%w to %s: %v", ErrNoDefaultRoute, f.dst.IP, err)
	}

	route = routes[0]

	if f.nic == "" {
		srcLink, err = netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			return nil, fmt.Errorf("%w by index %d: %v", ErrNoLink, route.LinkIndex, err)
		}

		f.nic = srcLink.Attrs().Name

		if f.src.MAC == nil {
			if linkAttrs := srcLink.Attrs(); linkAttrs.HardwareAddr != nil {
				f.src.MAC = linkAttrs.HardwareAddr
			}
		}
	} else {
		var err error

		srcLink, err = netlink.LinkByName(f.nic)
		if err != nil {
			return nil, fmt.Errorf("%w by name %s: %v", ErrNoLink, f.nic, err)
		}
	}

	if srcLink != nil {
		if f.src.MAC == nil {
			if linkAttrs := srcLink.Attrs(); linkAttrs.HardwareAddr != nil {
				f.src.MAC = linkAttrs.HardwareAddr
			}
		}

		if f.src.IP == nil {
			addrs, err := netlink.AddrList(srcLink, netlink.FAMILY_V4)
			if err != nil || len(addrs) == 0 {
				return nil, fmt.Errorf("%w for interface %s: %v", ErrNoIPv4Addr, f.nic, err)
			}

			// Prefer global unicast
			for _, addr := range addrs {
				if addr.IP.To4() != nil && addr.IP.IsGlobalUnicast() {
					f.src.IP = addr.IP

					break
				}
			}

			if f.src.IP == nil && len(addrs) > 0 {
				// Fallback to the first IP if no global unicast is found
				f.src.IP = addrs[0].IP
			}
		}
	}

	if f.dst.MAC == nil {
		// Determine dst.MAC (MAC of the next hop towards dstIP)
		var nextHopIP net.IP

		if route.Gw != nil {
			nextHopIP = route.Gw
		} else {
			// Destination is on the local network segment
			nextHopIP = f.dst.IP
		}

		neighbours, err := netlink.NeighList(srcLink.Attrs().Index, netlink.FAMILY_V4)
		if err != nil {
			return nil, fmt.Errorf("%w for interface %s to find MAC for %s: %v", ErrNoNeighbors, f.nic, nextHopIP.String(), err)
		}

		for _, neigh := range neighbours {
			if neigh.IP.Equal(nextHopIP) && neigh.HardwareAddr != nil {
				// Check for a valid and somewhat active ARP/neighbor state
				if isActiveNeighbour(neigh) {
					f.dst.MAC = neigh.HardwareAddr

					break
				}
			}
		}
	}

	if f.src.IP == nil {
		return nil, fmt.Errorf("source: %w", ErrNoIPv4Addr)
	}

	if f.src.MAC == nil {
		return nil, fmt.Errorf("source: %w", ErrNoMACAddr)
	}

	if f.dst.MAC == nil {
		return nil, fmt.Errorf("destination: %w", ErrNoDestMACAddr)
	}

	if !f.isValidIP(f.src.IP) {
		return nil, fmt.Errorf("source: %w", ErrInvalidIPAddr)
	}

	if !f.isValidMAC(f.src.MAC) {
		return nil, fmt.Errorf("source: %w", ErrInvalidMACAddr)
	}

	if !f.isValidIP(f.dst.IP) {
		return nil, fmt.Errorf("destination: %w", ErrInvalidIPAddr)
	}

	if !f.isValidMAC(f.dst.MAC) {
		return nil, fmt.Errorf("destination: %w", ErrInvalidMACAddr)
	}

	f.serializeBuf = gopacket.NewSerializeBuffer()
	f.serializeOpts = gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	f.layer.Ethernet = &layers.Ethernet{
		SrcMAC:       f.src.MAC,
		DstMAC:       f.dst.MAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	f.layer.IPv4 = &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Id:       0,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    f.src.IP.To4(),
		DstIP:    f.dst.IP.To4(),
		Flags:    layers.IPv4DontFragment,
	}
	f.layer.UDP = &layers.UDP{
		SrcPort: 0,
		DstPort: 53,
	}

	linkIdx := f.linkIndex()
	if linkIdx < 0 {
		return nil, fmt.Errorf("%w index for interface %s", ErrNoLink, f.nic)
	}

	program, err := xdp.NewProgram(f.queueID + 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create XDP program: %v", err)
	}
	f.program = program

	if err := program.Attach(linkIdx); err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("failed to attach XDP program to interface: %v", err)
	}

	// xskOpts := &xdp.SocketOptions{
	// 	NumFrames:              4096,
	// 	FrameSize:              2048,
	// 	FillRingNumDescs:       2048,
	// 	CompletionRingNumDescs: 2048,
	// 	RxRingNumDescs:         64,
	// 	TxRingNumDescs:         2048,
	// }

	xskOpts := &xdp.DefaultSocketOptions
	if f.socketOpts != nil {
		xskOpts = f.socketOpts
	}

	xsk, err := xdp.NewSocket(linkIdx, f.queueID, xskOpts)
	if err != nil {
		_ = f.Close()

		return nil, err
	}
	f.socket = xsk

	if err := f.program.Register(f.queueID, xsk.FD()); err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("failed to register socket in BPF map: %v", err)
	}

	if f.socket == nil || (f.socket != nil && f.socket.FD() < 0) {
		return nil, ErrInvalidXDPSocket
	}

	defaultMaxBatchSize := f.socket.NumFreeTxSlots()
	if f.maxBatchSize <= 0 || f.maxBatchSize > defaultMaxBatchSize {
		f.maxBatchSize = defaultMaxBatchSize
	}

	if !f.cacheDisabled {
		f.cache, err = otter.MustBuilder[*dns.Msg, *dns.Msg](f.cacheCapacity).
			WithTTL(f.cacheTTL).
			Build()

		if err != nil {
			_ = f.Close()

			return nil, fmt.Errorf("failed to create cache: %v", err)
		}
	}

	return f, nil
}

// Stats returns statistics about the XDP socket's ring queue usage and kernel
// interactions.
// This information can be useful for monitoring the performance and health of
// the XDP socket and the underlying network interface.
// The statistics are returned as an [xdp.Stats] struct, which contains various
// metrics related to the XDP socket's operation.
func (f *FastDNS) Stats() (xdp.Stats, error) {
	return f.socket.Stats()
}

// Close cleans up XDP sockets and programs.
// It is essential to call Close when you are done using [FastDNS] to prevent
// resource leaks.
// Failing to close may leave file descriptors, network handles, and kernel
// resources open, which can eventually exhaust system  resources, cause
// failures in network operations, or prevent new [FastDNS] instances from being
// created.
func (f *FastDNS) Close() error {
	var errors []error

	linkIdx := f.linkIndex()

	// f.program.Unregister(f.queueID)

	if f.socket != nil {
		if err := f.socket.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close XDP socket: %v", err))
		}
		f.socket = nil
	}

	if f.program != nil && linkIdx >= 0 {
		if err := f.program.Detach(linkIdx); err != nil {
			errors = append(errors, fmt.Errorf("failed to detach XDP program from interface: %v", err))
		}
	}

	if f.program != nil {
		if err := f.program.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close XDP program: %v", err))
		}
		f.program = nil
	}

	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// Query sends a DNS query using the XDP socket and returns the response.
// It serializes the DNS message, transmits it, and waits for a reply.
// If caching is enabled, it checks for a cached response before sending.
// On success, it returns the DNS response and round-trip time (RTT).
// On failure or timeout, it returns an error in the [Response] struct.
func (f *FastDNS) Query(msg *dns.Msg) *Response {
	var resp *Response

	for i := range f.maxRetries {
		resp = f.send(msg)
		if resp.IsError() {
			if i == f.maxRetries-1 {
				return resp
			}

			continue
		}

		if resp.IsSuccess() {
			return resp
		}
	}

	resp.Error = ErrNoDNSResponse

	return resp
}

// Queries sends multiple DNS queries and returns their responses.
// It serializes each DNS message, transmits them, and waits for replies.
// If caching is enabled, it checks for cached responses before sending.
// On success, it returns a slice of [Response] structs with the DNS responses
// and their round-trip times (RTT).
// On failure or timeout, it returns an error.
func (f *FastDNS) Queries(msgs ...*dns.Msg) ([]*Response, error) {
	if len(msgs) == 0 {
		return nil, ErrNoQueries
	}

	pool := pond.NewResultPool[*Response](f.maxBatchSize)
	group := pool.NewGroup()

	for _, msg := range msgs {
		group.Submit(func() *Response {
			return f.Query(msg)
		})
	}

	results, err := group.Wait()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// QueryFromFile reads entries from a file and performs DNS queries using a
// custom mapper function. Each line in the file is processed by the mapper
// function to create DNS messages.
//
// The mapper function takes a string from the file and returns a [dns.Msg].
// This allows for custom DNS message creation based on the file content.
// If the mapper returns nil, that entry is skipped.
func (f *FastDNS) QueryFromFile(file *os.File, mapper func(s string) *dns.Msg) ([]*Response, error) {
	if file == nil {
		return nil, ErrNoFile
	}

	if mapper == nil {
		return nil, ErrNoMapperFunc
	}

	var msgs []*dns.Msg

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if msg := mapper(line); msg != nil {
			msgs = append(msgs, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	return f.Queries(msgs...)
}

func (f *FastDNS) pollAndReceive(now time.Time, resp *Response) {
	backoff := 100
	for timeout := now.Add(f.timeout); time.Now().Before(timeout); {
		numRx, _, err := f.socket.Poll(1)
		if err != nil {
			resp.Error = fmt.Errorf("%w: %v", ErrPoll, err)
		}

		if numRx > 0 {
			rxDescs := f.socket.Receive(numRx)
			for i := range rxDescs {
				udpPacket, err := f.getUDPPacket(rxDescs[i])
				if err != nil {
					continue
				}

				dnsMsg := &dns.Msg{}
				if err := dnsMsg.Unpack(udpPacket.Payload); err != nil {
					continue
				}

				if resp.Message != nil {
					// Already received a response for this ID
					continue
				}

				msg := resp.query

				if !f.isTXMatch(msg, dnsMsg) {
					continue
				}

				if !f.cacheDisabled {
					f.cacheGetOrSet(msg, dnsMsg)
				}

				f.socket.Fill(rxDescs)

				resp.Message = dnsMsg
				resp.RTT = time.Since(now)

				return
			}

			f.socket.Fill(rxDescs)
		} else {
			runtime.Gosched()
			if backoff < defaultMaxBackoff {
				backoff = backoff * 3 / 2
			}

			time.Sleep(time.Duration(backoff) * time.Nanosecond)
		}
	}
}

func (f *FastDNS) send(msg *dns.Msg) *Response {
	if msg == nil {
		return nil
	}

	if msg.Id == 0 {
		msg.Id = dns.Id()
	}

	resp := &Response{RTT: -1, Message: nil, Error: nil, query: msg}

	payload, err := msg.Pack()
	if err != nil {
		resp.Error = fmt.Errorf("could not pack DNS message: %v", err)

		return resp
	}

	if !f.cacheDisabled {
		if cached, ok := f.cacheGetOrSet(msg, nil); ok {
			resp.Message = cached
			resp.RTT = 0

			return resp
		}
	}

	packetBytes, err := f.serializePayload(payload)
	if err != nil {
		resp.Error = fmt.Errorf("could not serialize payload: %v", err)

		return resp
	}

	var txDescs []xdp.Desc

	now := time.Now()
	backoff := 100

	for timeout := now.Add(f.timeout); now.Before(timeout); {
		txDescs, err = f.getTXDescs(packetBytes)
		if err != nil {
			resp.Error = err

			runtime.Gosched()
			if backoff < defaultMaxBackoff {
				backoff = backoff * 3 / 2
			}

			time.Sleep(time.Duration(backoff) * time.Nanosecond)

			continue
		}

		break
	}

	if err != nil {
		resp.Error = err

		return resp
	}

	// Send the query
	f.socket.Transmit(txDescs)

	if n := f.socket.NumFreeFillSlots(); n > 0 {
		f.socket.Fill(f.socket.GetDescs(n, true))
	}

	f.pollAndReceive(now, resp)

	if !resp.IsSuccess() {
		resp.Error = ErrNoDNSResponse
	}

	return resp
}

func (f *FastDNS) isValidMAC(mac net.HardwareAddr) bool {
	if mac == nil {
		return false
	}

	if len(mac) != 6 {
		return false
	}

	if mac[0] == 0x00 && mac[1] == 0x00 && mac[2] == 0x00 && mac[3] == 0x00 && mac[4] == 0x00 && mac[5] == 0x00 {
		return false
	}

	return true
}

func (f *FastDNS) isValidIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.To4() == nil {
		return false
	}

	if !f.allowNonGlobalIPs {
		if ip.IsLoopback() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsInterfaceLocalMulticast() {
			return false
		}
	}

	return true
}

func (f *FastDNS) isTXMatch(msg, resp *dns.Msg) bool {
	if msg == nil || resp == nil {
		return false
	}

	if msg.Id != resp.Id {
		return false
	}

	if len(msg.Question) != len(resp.Question) {
		return false
	}

	return resp.Question[0] == msg.Question[0]
}

func (f *FastDNS) cacheGetOrSet(key, value *dns.Msg) (*dns.Msg, bool) {
	if cached, ok := f.cache.Get(key); ok {
		return cached, ok
	}

	if value == nil {
		return nil, false
	}

	ok := f.cache.Set(key, value)

	return value, ok
}

func (f *FastDNS) serializePayload(payload []byte) ([]byte, error) {
	// Use a new buffer for each call (thread-safe)
	buf := gopacket.NewSerializeBuffer()

	f.layer.IPv4.Id++

	err := f.layer.UDP.SetNetworkLayerForChecksum(f.layer.IPv4)
	if err != nil {
		return nil, fmt.Errorf("could not set network layer for checksum: %v", err)
	}

	err = gopacket.SerializeLayers(
		buf,
		f.serializeOpts,
		f.layer.Ethernet,
		f.layer.IPv4,
		f.layer.UDP,
		gopacket.Payload(payload),
	)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (f *FastDNS) getTXDescs(packetBytes []byte) ([]xdp.Desc, error) {
	txDescs := f.socket.GetDescs(1, false)
	if len(txDescs) == 0 {
		return nil, ErrNoTXDescriptors
	}

	frame := f.socket.GetFrame(txDescs[0])
	frameLen := copy(frame, packetBytes)
	txDescs[0].Len = uint32(frameLen)

	return txDescs, nil
}

func (f *FastDNS) getUDPPacket(rxDesc xdp.Desc) (*layers.UDP, error) {
	minPacketSize := f.layer.Ethernet.Length + f.layer.IPv4.Length + f.layer.UDP.Length
	if rxDesc.Len < uint32(minPacketSize) {
		return nil, fmt.Errorf("%w: %d bytes, expected at least %d bytes", ErrPacketTooSmall, rxDesc.Len, minPacketSize)
	}

	pktData := f.socket.GetFrame(rxDesc)
	packet := gopacket.NewPacket(pktData[:rxDesc.Len], layers.LayerTypeEthernet, gopacket.Default)

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return nil, ErrNoIPLayer
	}
	ipPacket, _ := ipLayer.(*layers.IPv4)

	if !ipPacket.DstIP.Equal(f.src.IP) {
		return nil, ErrInvalidPacketDestination
	}

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return nil, ErrNoUDPLayer
	}
	udpPacket, _ := udpLayer.(*layers.UDP)

	if udpPacket.SrcPort != 53 {
		return nil, ErrInvalidPacketDestination
	}

	return udpPacket, nil
}

func (f *FastDNS) linkIndex() int {
	link, err := netlink.LinkByName(f.nic)
	if err != nil {
		return -1
	}

	return link.Attrs().Index
}
