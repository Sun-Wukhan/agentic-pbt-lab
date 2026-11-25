package harness

import (
	"net/netip"
	"testing"

	"go4.org/netipx"
	"pgregory.net/rapid"
	"tailscale.com/net/packet"
	"tailscale.com/types/ipproto"
	"tailscale.com/types/views"
	"tailscale.com/wgengine/filter"
	"tailscale.com/wgengine/filter/filtertype"
)

// P1: Parsed.Decode never panics on arbitrary bytes.
func TestPBTPacketDecodeNeverPanics(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		b := rapid.SliceOfN(rapid.Byte(), 0, 128).Draw(rt, "bytes")
		var p packet.Parsed
		p.Decode(b) // a panic here fails the property via rapid
		_ = p.String()
	})
}

// P2: Decode never panics on any truncation of a valid generated packet.
func TestPBTPacketDecodeTruncationsNeverPanic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		full := genUDP4Packet(rt)
		n := rapid.IntRange(0, len(full)).Draw(rt, "trunc")
		var p packet.Parsed
		p.Decode(full[:n])
	})
}

func genUDP4Packet(rt *rapid.T) []byte {
	src := netip.AddrFrom4([4]byte(rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "src")))
	dst := netip.AddrFrom4([4]byte(rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "dst")))
	payload := rapid.SliceOfN(rapid.Byte(), 0, 64).Draw(rt, "payload")
	h := packet.UDP4Header{
		IP4Header: packet.IP4Header{IPID: rapid.Uint16().Draw(rt, "ipid"), Src: src, Dst: dst},
		SrcPort:   rapid.Uint16().Draw(rt, "sport"),
		DstPort:   rapid.Uint16().Draw(rt, "dport"),
	}
	return packet.Generate(h, payload)
}

// P3: Generate(UDP4Header) -> Decode round-trips version, protocol,
// addresses, ports and payload.
func TestPBTPacketUDP4RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		src := netip.AddrFrom4([4]byte(rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "src")))
		dst := netip.AddrFrom4([4]byte(rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "dst")))
		sport := rapid.Uint16().Draw(rt, "sport")
		dport := rapid.Uint16().Draw(rt, "dport")
		payload := rapid.SliceOfN(rapid.Byte(), 0, 64).Draw(rt, "payload")
		h := packet.UDP4Header{
			IP4Header: packet.IP4Header{Src: src, Dst: dst},
			SrcPort:   sport, DstPort: dport,
		}
		buf := packet.Generate(h, payload)
		var p packet.Parsed
		p.Decode(buf)
		if p.IPVersion != 4 || p.IPProto != ipproto.UDP {
			rt.Fatalf("decoded version/proto = %v/%v; want 4/UDP", p.IPVersion, p.IPProto)
		}
		if p.Src != netip.AddrPortFrom(src, sport) || p.Dst != netip.AddrPortFrom(dst, dport) {
			rt.Fatalf("decoded %v > %v; want %v > %v", p.Src, p.Dst, netip.AddrPortFrom(src, sport), netip.AddrPortFrom(dst, dport))
		}
		if string(p.Payload()) != string(payload) {
			rt.Fatalf("payload mismatch: got %q want %q", p.Payload(), payload)
		}
	})
}

// P4: Generate(IP4Header with a non-TCP/UDP/ICMP protocol) -> Decode
// round-trips version, protocol and addresses.
func TestPBTPacketIP4RoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		src := netip.AddrFrom4([4]byte(rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "src")))
		dst := netip.AddrFrom4([4]byte(rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "dst")))
		proto := ipproto.Proto(rapid.ByteRange(200, 253).Draw(rt, "proto"))
		payload := rapid.SliceOfN(rapid.Byte(), 0, 32).Draw(rt, "payload")
		buf := packet.Generate(packet.IP4Header{IPProto: proto, Src: src, Dst: dst}, payload)
		var p packet.Parsed
		p.Decode(buf)
		if p.IPVersion != 4 || p.IPProto != proto || p.Src.Addr() != src || p.Dst.Addr() != dst {
			rt.Fatalf("decoded %v %v %v>%v; want 4 %v %v>%v", p.IPVersion, p.IPProto, p.Src.Addr(), p.Dst.Addr(), proto, src, dst)
		}
	})
}

// P5: Decode is deterministic: decoding the same bytes twice yields
// identical exported fields.
func TestPBTPacketDecodeDeterministic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		b := rapid.SliceOfN(rapid.Byte(), 0, 128).Draw(rt, "bytes")
		var p1, p2 packet.Parsed
		p1.Decode(b)
		p2.Decode(b)
		if p1.IPVersion != p2.IPVersion || p1.IPProto != p2.IPProto || p1.Src != p2.Src || p1.Dst != p2.Dst || p1.TCPFlags != p2.TCPFlags {
			rt.Fatalf("non-deterministic decode: %+v vs %+v", p1, p2)
		}
	})
}

// --- wgengine/filter ---

type filterRule struct {
	proto ipproto.Proto
	src   netip.Prefix
	dst   netip.Prefix
	first uint16
	last  uint16
}

func genFilterAddr(rt *rapid.T, label string) netip.Addr {
	// Ordinary unicast IPv4 in 10.0.0.0/8-ish space: avoids link-local,
	// multicast and other special-case paths in the filter's pre-checks.
	b := rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, label)
	return netip.AddrFrom4([4]byte{10, b[1] % 4, b[2], b[3]})
}

func genFilterRule(rt *rapid.T) filterRule {
	proto := rapid.SampledFrom([]ipproto.Proto{ipproto.TCP, ipproto.UDP}).Draw(rt, "proto")
	srcA := genFilterAddr(rt, "rsrc")
	dstA := genFilterAddr(rt, "rdst")
	first := rapid.Uint16Range(0, 1024).Draw(rt, "first")
	last := first + rapid.Uint16Range(0, 1024).Draw(rt, "span")
	return filterRule{
		proto: proto,
		src:   netip.PrefixFrom(srcA, int(rapid.IntRange(8, 32).Draw(rt, "sbits"))).Masked(),
		dst:   netip.PrefixFrom(dstA, int(rapid.IntRange(8, 32).Draw(rt, "dbits"))).Masked(),
		first: first, last: last,
	}
}

func buildFilter(rules []filterRule) *filter.Filter {
	ms := make([]filter.Match, 0, len(rules))
	for _, r := range rules {
		ms = append(ms, filter.Match{
			IPProto: views.SliceOf([]ipproto.Proto{r.proto}),
			Srcs:    []netip.Prefix{r.src},
			Dsts: []filtertype.NetPortRange{
				{Net: r.dst, Ports: filtertype.PortRange{First: r.first, Last: r.last}},
			},
		})
	}
	// localNets must cover the generated destination space: runIn4/6
	// drop any packet whose destination is not a local address
	// ("destination not allowed"), regardless of the match rules.
	var sb netipx.IPSetBuilder
	sb.AddPrefix(netip.MustParsePrefix("10.0.0.0/8"))
	local, err := sb.IPSet()
	if err != nil {
		panic(err)
	}
	return filter.New(ms, nil, local, nil, nil, func(string, ...any) {})
}

func modelAccept(rules []filterRule, proto ipproto.Proto, src, dst netip.Addr, port uint16) bool {
	for _, r := range rules {
		if r.proto == proto && r.src.Contains(src) && r.dst.Contains(dst) && port >= r.first && port <= r.last {
			return true
		}
	}
	return false
}

// F1: Filter.Check agrees with a naive reference model over the same
// rule set (TCP/UDP, IPv4 unicast).
func TestPBTFilterCheckVsModel(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(0, 5).Draw(rt, "nrules")
		rules := make([]filterRule, 0, n)
		for i := 0; i < n; i++ {
			rules = append(rules, genFilterRule(rt))
		}
		f := buildFilter(rules)
		proto := rapid.SampledFrom([]ipproto.Proto{ipproto.TCP, ipproto.UDP}).Draw(rt, "qproto")
		src := genFilterAddr(rt, "qsrc")
		dst := genFilterAddr(rt, "qdst")
		port := rapid.Uint16Range(0, 2048).Draw(rt, "qport")
		got := f.Check(src, dst, port, proto)
		wantAccept := modelAccept(rules, proto, src, dst, port)
		if wantAccept && got.IsDrop() {
			rt.Fatalf("Check(%v,%v,%v,%v) = %v; model says Accept (rules=%v)", src, dst, port, proto, got, rules)
		}
		if !wantAccept && !got.IsDrop() {
			rt.Fatalf("Check(%v,%v,%v,%v) = %v; model says Drop (rules=%v)", src, dst, port, proto, got, rules)
		}
	})
}

// F2: with no rules at all, everything is dropped.
func TestPBTFilterEmptyDropsAll(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		f := buildFilter(nil)
		src := genFilterAddr(rt, "src")
		dst := genFilterAddr(rt, "dst")
		port := rapid.Uint16().Draw(rt, "port")
		if got := f.Check(src, dst, port, ipproto.TCP); !got.IsDrop() {
			rt.Fatalf("empty filter Check = %v; want Drop", got)
		}
	})
}

// F3: mismatched address families are dropped (documented in Check:
// "Mismatched address families, no filters will match").
func TestPBTFilterMismatchedFamiliesDrop(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		rules := []filterRule{{
			proto: ipproto.TCP,
			src:   netip.MustParsePrefix("0.0.0.0/0"),
			dst:   netip.MustParsePrefix("0.0.0.0/0"),
			first: 0, last: 65535,
		}, {
			proto: ipproto.TCP,
			src:   netip.MustParsePrefix("::/0"),
			dst:   netip.MustParsePrefix("::/0"),
			first: 0, last: 65535,
		}}
		f := buildFilter(rules)
		v4 := genFilterAddr(rt, "v4")
		v6b := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(rt, "v6")
		var a16 [16]byte
		copy(a16[:], v6b)
		v6 := netip.AddrFrom16(a16)
		port := rapid.Uint16().Draw(rt, "port")
		if got := f.Check(v4, v6, port, ipproto.TCP); !got.IsDrop() {
			rt.Fatalf("Check(v4 %v, v6 %v) = %v; want Drop for mismatched families", v4, v6, got)
		}
		if got := f.Check(v6, v4, port, ipproto.TCP); !got.IsDrop() {
			rt.Fatalf("Check(v6 %v, v4 %v) = %v; want Drop for mismatched families", v6, v4, got)
		}
	})
}
