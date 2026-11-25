package harness

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math/big"
	"net/netip"
	"strings"
	"testing"

	"go4.org/mem"
	"pgregory.net/rapid"
	"tailscale.com/derp"
	"tailscale.com/disco"
	"tailscale.com/types/key"
	"tailscale.com/util/cmpver"
	"tailscale.com/util/dnsname"
)

// --- disco ---

func genNodeKey(rt *rapid.T) key.NodePublic {
	if rapid.Bool().Draw(rt, "zeroKey") {
		return key.NodePublic{}
	}
	b := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(rt, "key")
	return key.NodePublicFromRaw32(mem.B(b))
}

// D1: Ping AppendMarshal -> Parse round-trips TxID and NodeKey.
func TestPBTDiscoPingRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		var tx [12]byte
		copy(tx[:], rapid.SliceOfN(rapid.Byte(), 12, 12).Draw(rt, "tx"))
		m := &disco.Ping{TxID: tx, NodeKey: genNodeKey(rt)}
		wire := m.AppendMarshal(nil)
		got, err := disco.Parse(wire)
		if err != nil {
			rt.Fatalf("Parse(marshaled Ping) error: %v", err)
		}
		p, ok := got.(*disco.Ping)
		if !ok {
			rt.Fatalf("Parse returned %T; want *disco.Ping", got)
		}
		if p.TxID != tx || p.NodeKey != m.NodeKey {
			rt.Fatalf("round trip mismatch: got tx=%x key=%v; want tx=%x key=%v", p.TxID, p.NodeKey, tx, m.NodeKey)
		}
	})
}

// D2: Pong AppendMarshal -> Parse round-trips TxID and Src.
func TestPBTDiscoPongRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		var tx [12]byte
		copy(tx[:], rapid.SliceOfN(rapid.Byte(), 12, 12).Draw(rt, "tx"))
		src := genAddr(rt)
		port := rapid.Uint16().Draw(rt, "port")
		m := &disco.Pong{TxID: tx, Src: netip.AddrPortFrom(src, port)}
		wire := m.AppendMarshal(nil)
		got, err := disco.Parse(wire)
		if err != nil {
			rt.Fatalf("Parse(marshaled Pong) error: %v", err)
		}
		p, ok := got.(*disco.Pong)
		if !ok {
			rt.Fatalf("Parse returned %T; want *disco.Pong", got)
		}
		if p.TxID != tx || p.Src != m.Src {
			rt.Fatalf("round trip mismatch: got %v %v; want %v %v", p.TxID, p.Src, tx, m.Src)
		}
	})
}

// D3: CallMeMaybe AppendMarshal -> Parse round-trips the endpoint list.
func TestPBTDiscoCallMeMaybeRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 5).Draw(rt, "n")
		eps := make([]netip.AddrPort, 0, n)
		for i := 0; i < n; i++ {
			eps = append(eps, netip.AddrPortFrom(genAddr(rt), rapid.Uint16().Draw(rt, "port")))
		}
		m := &disco.CallMeMaybe{MyNumber: eps}
		wire := m.AppendMarshal(nil)
		got, err := disco.Parse(wire)
		if err != nil {
			rt.Fatalf("Parse(marshaled CallMeMaybe) error: %v", err)
		}
		c, ok := got.(*disco.CallMeMaybe)
		if !ok {
			rt.Fatalf("Parse returned %T; want *disco.CallMeMaybe", got)
		}
		if len(c.MyNumber) != len(eps) {
			rt.Fatalf("got %d endpoints; want %d (%v)", len(c.MyNumber), len(eps), eps)
		}
		for i := range eps {
			if c.MyNumber[i] != eps[i] {
				rt.Fatalf("endpoint %d: got %v; want %v", i, c.MyNumber[i], eps[i])
			}
		}
	})
}

// D4: disco.Parse never panics on arbitrary bytes, nor on any
// truncation of a valid message.
func TestPBTDiscoParseNeverPanics(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		b := rapid.SliceOfN(rapid.Byte(), 0, 96).Draw(rt, "bytes")
		_, _ = disco.Parse(b)
		var tx [12]byte
		copy(tx[:], rapid.SliceOfN(rapid.Byte(), 12, 12).Draw(rt, "tx"))
		wire := (&disco.Ping{TxID: tx, NodeKey: genNodeKey(rt)}).AppendMarshal(nil)
		n := rapid.IntRange(0, len(wire)).Draw(rt, "trunc")
		_, _ = disco.Parse(wire[:n])
	})
}

// --- derp ---

// R1: a manually encoded frame header (type byte + big-endian uint32
// length) is read back exactly by derp.ReadFrameHeader.
func TestPBTDerpFrameHeaderRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		typ := rapid.Byte().Draw(rt, "type")
		l := rapid.Uint32().Draw(rt, "len")
		var hdr [derp.FrameHeaderLen]byte
		hdr[0] = typ
		binary.BigEndian.PutUint32(hdr[1:], l)
		br := bufio.NewReader(bytes.NewReader(hdr[:]))
		gotT, gotL, err := derp.ReadFrameHeader(br)
		if err != nil {
			rt.Fatalf("ReadFrameHeader(%x) error: %v", hdr, err)
		}
		if byte(gotT) != typ || gotL != l {
			rt.Fatalf("ReadFrameHeader = %v,%v; want %v,%v", gotT, gotL, typ, l)
		}
	})
}

// R2: derp.ReadFrameHeader never panics on arbitrary or truncated
// input, and reports an error for inputs shorter than a header.
func TestPBTDerpFrameHeaderNeverPanics(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		b := rapid.SliceOfN(rapid.Byte(), 0, 16).Draw(rt, "bytes")
		br := bufio.NewReader(bytes.NewReader(b))
		_, _, err := derp.ReadFrameHeader(br)
		if len(b) < derp.FrameHeaderLen && err == nil {
			rt.Fatalf("ReadFrameHeader(%d bytes) succeeded; want error", len(b))
		}
	})
}

// --- dnsname ---

func genLabel(rt *rapid.T, label string) string {
	n := rapid.IntRange(1, 70).Draw(rt, label+"len")
	return rapid.StringOfN(rapid.SampledFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_")), n, n, -1).Draw(rt, label)
}

// N1: a successful ToFQDN is a fixed point: it ends with a dot,
// re-parsing it succeeds and returns it unchanged, and its label
// count matches the input's structure.
func TestPBTDnsnameToFQDNFixedPoint(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 4).Draw(rt, "nlabels")
		labels := make([]string, 0, n)
		for i := 0; i < n; i++ {
			labels = append(labels, genLabel(rt, "label"))
		}
		name := strings.Join(labels, ".")
		if rapid.Bool().Draw(rt, "trailingDot") {
			name += "."
		}
		f, err := dnsname.ToFQDN(name)
		if err != nil {
			// Only acceptable if a label exceeds 63 bytes or the name
			// exceeds 254 (generators can produce both).
			long := false
			for _, l := range labels {
				if len(l) > 63 {
					long = true
				}
			}
			if !long && len(name) <= 253 {
				rt.Fatalf("ToFQDN(%q) error %v; labels within limits", name, err)
			}
			return
		}
		s := f.WithTrailingDot()
		if !strings.HasSuffix(s, ".") {
			rt.Fatalf("ToFQDN(%q) = %q; no trailing dot", name, s)
		}
		f2, err := dnsname.ToFQDN(s)
		if err != nil || f2 != f {
			rt.Fatalf("ToFQDN not a fixed point: %q -> %q -> %q (err %v)", name, s, f2, err)
		}
		if f.NumLabels() != n {
			rt.Fatalf("NumLabels(%q) = %d; want %d", s, f.NumLabels(), n)
		}
	})
}

// N2: ToFQDN rejects names containing whitespace/control characters,
// empty labels, labels over 63 bytes and names over 254 bytes.
func TestPBTDnsnameToFQDNRejects(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		kind := rapid.IntRange(0, 3).Draw(rt, "kind")
		var name string
		switch kind {
		case 0: // whitespace/control inside a label
			bad := rapid.SampledFrom([]byte{' ', '\t', '\n', '\r', 0x7f, 0x01}).Draw(rt, "bad")
			name = "goo" + string([]byte{bad}) + "d.example"
		case 1: // empty label
			name = "a..b"
			if rapid.Bool().Draw(rt, "lead") {
				name = "a...b"
			}
		case 2: // label of 64+ bytes
			name = strings.Repeat("a", int(rapid.IntRange(64, 80).Draw(rt, "l"))) + ".example"
		case 3: // total length over 254 with legal labels
			lbl := strings.Repeat("a", 60)
			name = strings.Join([]string{lbl, lbl, lbl, lbl, lbl}, ".")
		}
		if _, err := dnsname.ToFQDN(name); err == nil {
			rt.Fatalf("ToFQDN(%q) succeeded; want rejection (kind %d)", name, kind)
		}
	})
}

// N3: SanitizeLabel output is either empty or a valid label
// (ValidLabel accepts it), is lowercase, and sanitizing is idempotent.
func TestPBTDnsnameSanitizeLabel(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		in := rapid.String().Draw(rt, "in")
		out := dnsname.SanitizeLabel(in)
		if out == "" {
			return
		}
		if err := dnsname.ValidLabel(out); err != nil {
			rt.Fatalf("SanitizeLabel(%q) = %q, but ValidLabel rejects it: %v", in, out, err)
		}
		if out != strings.ToLower(out) {
			rt.Fatalf("SanitizeLabel(%q) = %q; not lowercase", in, out)
		}
		if again := dnsname.SanitizeLabel(out); again != out {
			rt.Fatalf("SanitizeLabel not idempotent: %q -> %q -> %q", in, out, again)
		}
	})
}

// N4: FQDN Parent/Contains agree with a naive label-list model.
func TestPBTDnsnameParentContains(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 5).Draw(rt, "nlabels")
		labels := make([]string, 0, n)
		for i := 0; i < n; i++ {
			labels = append(labels, rapid.StringOfN(rapid.SampledFrom([]rune("abcde")), 1, 8, -1).Draw(rt, "label"))
		}
		f, err := dnsname.ToFQDN(strings.Join(labels, "."))
		if err != nil {
			rt.Fatalf("ToFQDN: %v", err)
		}
		if !f.Contains(f) {
			rt.Fatalf("%q does not contain itself", f)
		}
		p := f.Parent()
		if n == 1 {
			if p != "" {
				rt.Fatalf("Parent(%q) = %q; want empty", f, p)
			}
			return
		}
		wantParent, _ := dnsname.ToFQDN(strings.Join(labels[1:], "."))
		if p != wantParent {
			rt.Fatalf("Parent(%q) = %q; want %q", f, p, wantParent)
		}
		if !p.Contains(f) {
			rt.Fatalf("parent %q does not contain %q", p, f)
		}
		if f.Contains(p) && p != f {
			rt.Fatalf("%q claims to contain its parent %q", f, p)
		}
	})
}

// --- cmpver ---

func genVersion(rt *rapid.T, label string) string {
	parts := rapid.SliceOfN(rapid.StringMatching(`[0-9a-zA-Z][0-9a-zA-Z.\-]*`), 1, 1).Draw(rt, label)
	return parts[0]
}

func sign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}

// C1: Compare is antisymmetric and reflexive, and Less/LessEq agree
// with it.
func TestPBTCmpverAntisymmetry(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		a := genVersion(rt, "a")
		b := genVersion(rt, "b")
		if sign(cmpver.Compare(a, b)) != -sign(cmpver.Compare(b, a)) {
			rt.Fatalf("Compare(%q,%q)=%d but Compare(%q,%q)=%d", a, b, cmpver.Compare(a, b), b, a, cmpver.Compare(b, a))
		}
		if cmpver.Compare(a, a) != 0 {
			rt.Fatalf("Compare(%q,%q) != 0", a, a)
		}
		if cmpver.Less(a, b) != (cmpver.Compare(a, b) < 0) || cmpver.LessEq(a, b) != (cmpver.Compare(a, b) <= 0) {
			rt.Fatalf("Less/LessEq disagree with Compare for %q,%q", a, b)
		}
	})
}

// C2: Compare induces a total order (transitivity over triples).
func TestPBTCmpverTransitivity(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		a, b, c := genVersion(rt, "a"), genVersion(rt, "b"), genVersion(rt, "c")
		ab, bc, ac := cmpver.Compare(a, b), cmpver.Compare(b, c), cmpver.Compare(a, c)
		if ab <= 0 && bc <= 0 && ac > 0 {
			rt.Fatalf("transitivity violated: %q<=%q<=%q but %q>%q", a, b, c, a, c)
		}
		if ab >= 0 && bc >= 0 && ac < 0 {
			rt.Fatalf("transitivity violated: %q>=%q>=%q but %q<%q", a, b, c, a, c)
		}
		if ab == 0 && bc == 0 && ac != 0 {
			rt.Fatalf("equality not transitive: %q==%q==%q but Compare(%q,%q)=%d", a, b, c, a, c, ac)
		}
	})
}

// C3: for dotted-numeric versions, Compare agrees with an independent
// model: component-wise big-integer comparison; if all shared
// components are equal, the version with more components is greater
// (documented: "12" is less than "12.0").
func TestPBTCmpverDottedNumericModel(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		gen := func(label string) (string, []*big.Int) {
			n := rapid.IntRange(1, 4).Draw(rt, label+"n")
			parts := make([]string, 0, n)
			nums := make([]*big.Int, 0, n)
			for i := 0; i < n; i++ {
				d := rapid.StringMatching(`[0-9]{1,25}`).Draw(rt, label+"d")
				parts = append(parts, d)
				v, _ := new(big.Int).SetString(d, 10)
				nums = append(nums, v)
			}
			return strings.Join(parts, "."), nums
		}
		as, an := gen("a")
		bs, bn := gen("b")
		want := 0
		for i := 0; i < len(an) && i < len(bn); i++ {
			if c := an[i].Cmp(bn[i]); c != 0 {
				want = c
				break
			}
		}
		if want == 0 && len(an) != len(bn) {
			if len(an) < len(bn) {
				want = -1
			} else {
				want = 1
			}
		}
		if got := sign(cmpver.Compare(as, bs)); got != want {
			rt.Fatalf("Compare(%q,%q) sign = %d; model = %d", as, bs, got, want)
		}
	})
}
