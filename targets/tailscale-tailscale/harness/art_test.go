package harness

import (
	"net/netip"
	"testing"

	"pgregory.net/rapid"
	"tailscale.com/net/art"
)

// Generators shared by the ART properties. Addresses and prefixes are
// drawn from small pools so collisions, overlaps and nesting are common.

func genAddr(rt *rapid.T) netip.Addr {
	return rapid.OneOf(
		rapid.Custom(func(rt *rapid.T) netip.Addr {
			b := rapid.SliceOfN(rapid.Byte(), 4, 4).Draw(rt, "v4bytes")
			return netip.AddrFrom4([4]byte(b))
		}),
		rapid.Custom(func(rt *rapid.T) netip.Addr {
			// IPv6 from a small set of /32-ish clusters to force overlap.
			hi := rapid.SampledFrom([]uint32{0x20010db8, 0x20010db9, 0xfd000000}).Draw(rt, "v6hi")
			lo := rapid.Uint32Range(0, 3).Draw(rt, "v6lo")
			var b [16]byte
			b[0], b[1], b[2], b[3] = byte(hi>>24), byte(hi>>16), byte(hi>>8), byte(hi)
			b[15] = byte(lo)
			return netip.AddrFrom16(b)
		}),
	).Draw(rt, "addr")
}

func genPrefix(rt *rapid.T) netip.Prefix {
	a := genAddr(rt)
	var bits int
	if a.Is4() {
		bits = rapid.IntRange(0, 32).Draw(rt, "bits")
	} else {
		bits = rapid.IntRange(0, 128).Draw(rt, "bits")
	}
	return netip.PrefixFrom(a, bits) // deliberately not always Masked
}

// modelLPM is the brute-force longest-prefix-match model: among all
// (masked) prefixes containing addr, the one with the most bits wins;
// its most recent value is the model's answer.
type modelEntry struct {
	pfx netip.Prefix
	val int
}

func modelGet(entries []modelEntry, addr netip.Addr) (int, bool) {
	best, bestBits := 0, -1
	found := false
	for _, e := range entries {
		if e.pfx.Contains(addr) && e.pfx.Bits() >= bestBits {
			best, bestBits, found = e.val, e.pfx.Bits(), true
		}
	}
	return best, found
}

func applyInsert(entries []modelEntry, pfx netip.Prefix, val int) []modelEntry {
	pfx = pfx.Masked()
	for i, e := range entries {
		if e.pfx == pfx {
			entries[i].val = val
			return entries
		}
	}
	return append(entries, modelEntry{pfx, val})
}

func applyDelete(entries []modelEntry, pfx netip.Prefix) []modelEntry {
	pfx = pfx.Masked()
	out := entries[:0]
	for _, e := range entries {
		if e.pfx != pfx {
			out = append(out, e)
		}
	}
	return out
}

// A1: after Insert(pfx, v), every address in pfx resolves to v unless a
// longer prefix also covers it (checked exactly via the model in A2;
// here the single-route case).
func TestPBTArtSingleInsertGet(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		pfx := genPrefix(rt)
		addr := genAddr(rt)
		var tbl art.Table[int]
		tbl.Insert(pfx, 42)
		got, ok := tbl.Get(addr)
		want, wantOK := modelGet([]modelEntry{{pfx.Masked(), 42}}, addr)
		if ok != wantOK || (ok && got != want) {
			rt.Fatalf("Get(%v) after Insert(%v,42) = %v,%v; model = %v,%v", addr, pfx, got, ok, want, wantOK)
		}
	})
}

// A2: a random sequence of inserts/deletes always agrees with the
// brute-force longest-prefix-match model.
func TestPBTArtModelVsSequence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(1, 30).Draw(rt, "nops")
		var tbl art.Table[int]
		var entries []modelEntry
		for i := 0; i < n; i++ {
			pfx := genPrefix(rt)
			if rapid.Bool().Draw(rt, "isInsert") {
				tbl.Insert(pfx, i)
				entries = applyInsert(entries, pfx, i)
			} else {
				tbl.Delete(pfx)
				entries = applyDelete(entries, pfx)
			}
			for j := 0; j < 3; j++ {
				addr := genAddr(rt)
				got, ok := tbl.Get(addr)
				want, wantOK := modelGet(entries, addr)
				if ok != wantOK || (ok && got != want) {
					rt.Fatalf("after %d ops, Get(%v) = %v,%v; model = %v,%v (entries=%v)", i+1, addr, got, ok, want, wantOK, entries)
				}
			}
		}
	})
}

// A3: inserting the same prefix twice overwrites (last write wins).
func TestPBTArtOverwriteLastWins(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		pfx := genPrefix(rt).Masked()
		var tbl art.Table[int]
		tbl.Insert(pfx, 1)
		tbl.Insert(pfx, 2)
		addr := pfx.Addr()
		got, ok := tbl.Get(addr)
		if !ok || got != 2 {
			rt.Fatalf("Get(%v) after overwrite of %v = %v,%v; want 2,true", addr, pfx, got, ok)
		}
	})
}

// A4: Insert masks its argument (documented in table.go: "pfx =
// pfx.Masked()"), so an unmasked prefix behaves exactly like its
// masked form, including for Delete.
func TestPBTArtInsertMasksPrefix(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		pfx := genPrefix(rt)
		var tbl art.Table[int]
		tbl.Insert(pfx, 7)
		addr := pfx.Masked().Addr()
		if got, ok := tbl.Get(addr); !ok || got != 7 {
			rt.Fatalf("Get(%v) after Insert(unmasked %v) = %v,%v; want 7,true", addr, pfx, got, ok)
		}
		tbl.Delete(pfx) // unmasked delete must also remove the masked route
		if _, ok := tbl.Get(addr); ok {
			rt.Fatalf("Get(%v) still present after Delete(unmasked %v)", addr, pfx)
		}
	})
}

// A5: family isolation — IPv4 routes never answer IPv6 lookups and
// vice versa.
func TestPBTArtFamilyIsolation(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		pfx := genPrefix(rt)
		addr := genAddr(rt)
		var tbl art.Table[int]
		tbl.Insert(pfx, 3)
		if pfx.Addr().Is4() != addr.Is4() {
			if _, ok := tbl.Get(addr); ok {
				rt.Fatalf("Get(%v) matched cross-family route %v", addr, pfx)
			}
		}
	})
}
