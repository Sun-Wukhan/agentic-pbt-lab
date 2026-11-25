package harness

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"go4.org/mem"
	"pgregory.net/rapid"
	"tailscale.com/tka"
	"tailscale.com/types/key"
)

// ---------------------------------------------------------------------------
// Wave 2, secondary surface: types/key round-trips and parser robustness.
// Keys are constructed from rapid-drawn raw bytes wherever a FromRaw32
// constructor exists, so failures replay deterministically.
// ---------------------------------------------------------------------------

func drawRaw32(rt *rapid.T, label string) [32]byte {
	b := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(rt, label)
	var out [32]byte
	copy(out[:], b)
	if out == ([32]byte{}) {
		out[0] = 1 // zero private keys panic in Public()/Shared()
	}
	return out
}

// K1 (wave 2): MarshalText/UnmarshalText round-trips for node, machine,
// disco and tailnet-lock keys; Public() is a deterministic function of the
// private key; NodePublic MarshalBinary round-trips; TL key IDs agree
// across key.TLPrivate, key.TLPublic and tka.Key.
func TestPBTKeyTextRoundTrips(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		raw := drawRaw32(rt, "raw")

		// NodePrivate / NodePublic.
		np := key.NodePrivateFromRaw32(mem.B(raw[:]))
		npText, err := np.MarshalText()
		if err != nil {
			rt.Fatalf("NodePrivate MarshalText: %v", err)
		}
		var np2 key.NodePrivate
		if err := np2.UnmarshalText(npText); err != nil {
			rt.Fatalf("NodePrivate UnmarshalText: %v", err)
		}
		if !np.Equal(np2) {
			rt.Fatalf("NodePrivate text round-trip mismatch")
		}
		if np.Public().Raw32() != np2.Public().Raw32() {
			rt.Fatalf("NodePrivate Public() changed across text round-trip")
		}
		if np.Public().Raw32() != np.Public().Raw32() {
			rt.Fatalf("NodePrivate Public() not deterministic")
		}
		nPub := np.Public()
		nPubText, err := nPub.MarshalText()
		if err != nil {
			rt.Fatalf("NodePublic MarshalText: %v", err)
		}
		var nPub2 key.NodePublic
		if err := nPub2.UnmarshalText(nPubText); err != nil {
			rt.Fatalf("NodePublic UnmarshalText: %v", err)
		}
		if nPub2.Raw32() != nPub.Raw32() {
			rt.Fatalf("NodePublic text round-trip mismatch")
		}
		nPubBin, err := nPub.MarshalBinary()
		if err != nil {
			rt.Fatalf("NodePublic MarshalBinary: %v", err)
		}
		var nPub3 key.NodePublic
		if err := nPub3.UnmarshalBinary(nPubBin); err != nil {
			rt.Fatalf("NodePublic UnmarshalBinary: %v", err)
		}
		if nPub3.Raw32() != nPub.Raw32() {
			rt.Fatalf("NodePublic binary round-trip mismatch")
		}

		// MachinePrivate / MachinePublic.
		var mp key.MachinePrivate
		if err := mp.UnmarshalText([]byte("privkey:" + hex.EncodeToString(raw[:]))); err != nil {
			rt.Fatalf("MachinePrivate UnmarshalText: %v", err)
		}
		mpText, err := mp.MarshalText()
		if err != nil {
			rt.Fatalf("MachinePrivate MarshalText: %v", err)
		}
		var mp2 key.MachinePrivate
		if err := mp2.UnmarshalText(mpText); err != nil {
			rt.Fatalf("MachinePrivate re-UnmarshalText: %v", err)
		}
		if !mp.Equal(mp2) {
			rt.Fatalf("MachinePrivate text round-trip mismatch")
		}
		mPub := key.MachinePublicFromRaw32(mem.B(raw[:]))
		mPubText, err := mPub.MarshalText()
		if err != nil {
			rt.Fatalf("MachinePublic MarshalText: %v", err)
		}
		var mPub2 key.MachinePublic
		if err := mPub2.UnmarshalText(mPubText); err != nil {
			rt.Fatalf("MachinePublic UnmarshalText: %v", err)
		}
		if !bytes.Equal(mPub2.UntypedBytes(), mPub.UntypedBytes()) {
			rt.Fatalf("MachinePublic text round-trip mismatch")
		}
		if mp.Public().UntypedBytes() == nil || !bytes.Equal(mp.Public().UntypedBytes(), mp2.Public().UntypedBytes()) {
			rt.Fatalf("MachinePrivate Public() changed across text round-trip")
		}

		// DiscoPrivate / DiscoPublic, plus X25519 shared-secret agreement.
		dp := key.DiscoPrivateFromRaw32(mem.B(raw[:]))
		rawB := drawRaw32(rt, "rawB")
		dpB := key.DiscoPrivateFromRaw32(mem.B(rawB[:]))
		if dp.Public().Raw32() != dp.Public().Raw32() {
			rt.Fatalf("DiscoPrivate Public() not deterministic")
		}
		dPub := dp.Public()
		dPubText, err := dPub.MarshalText()
		if err != nil {
			rt.Fatalf("DiscoPublic MarshalText: %v", err)
		}
		var dPub2 key.DiscoPublic
		if err := dPub2.UnmarshalText(dPubText); err != nil {
			rt.Fatalf("DiscoPublic UnmarshalText: %v", err)
		}
		if dPub2.Raw32() != dPub.Raw32() {
			rt.Fatalf("DiscoPublic text round-trip mismatch")
		}
		if !dp.Shared(dpB.Public()).Equal(dpB.Shared(dp.Public())) {
			rt.Fatalf("Disco Shared() not symmetric")
		}

		// TLPrivate / TLPublic (tailnet-lock keys), incl. the CLI text form.
		seed := drawRaw32(rt, "seed")
		edPriv := ed25519.NewKeyFromSeed(seed[:])
		var tl key.TLPrivate
		if err := tl.UnmarshalText([]byte("nlpriv:" + hex.EncodeToString(edPriv))); err != nil {
			rt.Fatalf("TLPrivate UnmarshalText: %v", err)
		}
		tlText, err := tl.MarshalText()
		if err != nil {
			rt.Fatalf("TLPrivate MarshalText: %v", err)
		}
		var tl2 key.TLPrivate
		if err := tl2.UnmarshalText(tlText); err != nil {
			rt.Fatalf("TLPrivate re-UnmarshalText: %v", err)
		}
		if !tl.Equal(tl2) {
			rt.Fatalf("TLPrivate text round-trip mismatch")
		}
		tlPub := tl.Public()
		if !tlPub.Equal(tl2.Public()) {
			rt.Fatalf("TLPrivate Public() changed across text round-trip")
		}
		tlPubText, err := tlPub.MarshalText()
		if err != nil {
			rt.Fatalf("TLPublic MarshalText: %v", err)
		}
		var tlPub2 key.TLPublic
		if err := tlPub2.UnmarshalText(tlPubText); err != nil {
			rt.Fatalf("TLPublic UnmarshalText: %v", err)
		}
		if !tlPub2.Equal(tlPub) {
			rt.Fatalf("TLPublic text round-trip mismatch")
		}
		var tlPub3 key.TLPublic
		if err := tlPub3.UnmarshalText([]byte(tlPub.CLIString())); err != nil {
			rt.Fatalf("TLPublic UnmarshalText(CLIString): %v", err)
		}
		if !tlPub3.Equal(tlPub) {
			rt.Fatalf("TLPublic CLI-form round-trip mismatch")
		}
		// KeyID agreement: TLPrivate.KeyID == TLPublic.KeyID == tka.Key.ID.
		tkaKey := tka.Key{Kind: tka.Key25519, Votes: 1, Public: edPriv.Public().(ed25519.PublicKey)}
		tkaID, err := tkaKey.ID()
		if err != nil {
			rt.Fatalf("tka.Key ID: %v", err)
		}
		if !bytes.Equal(tl.KeyID(), tkaID) || !bytes.Equal(tlPub.KeyID(), tkaID) {
			rt.Fatalf("TL KeyID disagreement: tl=%x tlpub=%x tka=%x", tl.KeyID(), tlPub.KeyID(), tkaID)
		}
	})
}

// K2 (wave 2): key text/binary parsers never panic on arbitrary or mutated
// input; anything they accept re-marshals to a stable, re-parseable form.
func TestPBTKeyParsersNeverPanic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		raw := drawRaw32(rt, "raw")
		validTexts := map[string][]byte{}
		if b, err := key.NodePublicFromRaw32(mem.B(raw[:])).MarshalText(); err == nil {
			validTexts["nodepub"] = b
		}
		if b, err := key.MachinePublicFromRaw32(mem.B(raw[:])).MarshalText(); err == nil {
			validTexts["machinepub"] = b
		}
		if b, err := key.DiscoPublicFromRaw32(mem.B(raw[:])).MarshalText(); err == nil {
			validTexts["discopub"] = b
		}

		var input []byte
		if rapid.Bool().Draw(rt, "mutate") && len(validTexts) > 0 {
			keys := make([]string, 0, len(validTexts))
			for k := range validTexts {
				keys = append(keys, k)
			}
			input = append([]byte(nil), validTexts[keys[rapid.IntRange(0, len(keys)-1).Draw(rt, "which")]]...)
			switch rapid.IntRange(0, 2).Draw(rt, "mode") {
			case 0:
				if len(input) > 0 {
					input[rapid.IntRange(0, len(input)-1).Draw(rt, "pos")] ^= byte(rapid.IntRange(1, 255).Draw(rt, "xor"))
				}
			case 1:
				input = input[:rapid.IntRange(0, len(input)).Draw(rt, "trunc")]
			case 2:
				input = append(input, rapid.SliceOfN(rapid.Byte(), 1, 8).Draw(rt, "tail")...)
			}
		} else {
			input = rapid.SliceOfN(rapid.Byte(), 0, 96).Draw(rt, "bytes")
		}

		// Every parser: no panic; on success the value must be canonical
		// (marshal -> parse -> equal).
		var np key.NodePrivate
		if err := np.UnmarshalText(input); err == nil {
			b, err := np.MarshalText()
			if err != nil {
				rt.Fatalf("NodePrivate re-marshal after parse: %v", err)
			}
			var np2 key.NodePrivate
			if err := np2.UnmarshalText(b); err != nil || !np.Equal(np2) {
				rt.Fatalf("NodePrivate accepted non-canonical input %q", input)
			}
		}
		var nPub key.NodePublic
		if err := nPub.UnmarshalText(input); err == nil {
			b, _ := nPub.MarshalText()
			var nPub2 key.NodePublic
			if err := nPub2.UnmarshalText(b); err != nil || nPub2.Raw32() != nPub.Raw32() {
				rt.Fatalf("NodePublic accepted non-canonical input %q", input)
			}
		}
		var mp key.MachinePrivate
		if err := mp.UnmarshalText(input); err == nil {
			b, _ := mp.MarshalText()
			var mp2 key.MachinePrivate
			if err := mp2.UnmarshalText(b); err != nil || !mp.Equal(mp2) {
				rt.Fatalf("MachinePrivate accepted non-canonical input %q", input)
			}
		}
		var mPub key.MachinePublic
		if err := mPub.UnmarshalText(input); err == nil {
			b, _ := mPub.MarshalText()
			var mPub2 key.MachinePublic
			if err := mPub2.UnmarshalText(b); err != nil || !bytes.Equal(mPub2.UntypedBytes(), mPub.UntypedBytes()) {
				rt.Fatalf("MachinePublic accepted non-canonical input %q", input)
			}
		}
		var dPub key.DiscoPublic
		if err := dPub.UnmarshalText(input); err == nil {
			b, _ := dPub.MarshalText()
			var dPub2 key.DiscoPublic
			if err := dPub2.UnmarshalText(b); err != nil || dPub2.Raw32() != dPub.Raw32() {
				rt.Fatalf("DiscoPublic accepted non-canonical input %q", input)
			}
		}
		var tl key.TLPrivate
		if err := tl.UnmarshalText(input); err == nil {
			b, _ := tl.MarshalText()
			var tl2 key.TLPrivate
			if err := tl2.UnmarshalText(b); err != nil || !tl.Equal(tl2) {
				rt.Fatalf("TLPrivate accepted non-canonical input %q", input)
			}
		}
		var tlPub key.TLPublic
		if err := tlPub.UnmarshalText(input); err == nil {
			b, _ := tlPub.MarshalText()
			var tlPub2 key.TLPublic
			if err := tlPub2.UnmarshalText(b); err != nil || !tlPub.Equal(tlPub2) {
				rt.Fatalf("TLPublic accepted non-canonical input %q", input)
			}
		}
		var nPubB key.NodePublic
		_ = nPubB.UnmarshalBinary(input) // must not panic at any length
	})
}

// K3 (wave 2): public-key text parsers reject each other's prefixes — a
// nodekey: string is not an mkey:, discokey: or nlpub:, and so on. (Private
// node/machine keys intentionally share the privkey: format, so that pair
// is excluded by design.)
func TestPBTKeyCrossPrefixRejection(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		raw := drawRaw32(rt, "raw")
		nodeText, _ := key.NodePublicFromRaw32(mem.B(raw[:])).MarshalText()
		machineText, _ := key.MachinePublicFromRaw32(mem.B(raw[:])).MarshalText()
		discoText, _ := key.DiscoPublicFromRaw32(mem.B(raw[:])).MarshalText()
		seed := drawRaw32(rt, "seed")
		var tl key.TLPrivate
		if err := tl.UnmarshalText([]byte("nlpriv:" + hex.EncodeToString(ed25519.NewKeyFromSeed(seed[:])))); err != nil {
			rt.Fatalf("TLPrivate UnmarshalText: %v", err)
		}
		tlText, _ := tl.Public().MarshalText()

		texts := map[string][]byte{
			"node": nodeText, "machine": machineText, "disco": discoText, "tl": tlText,
		}
		parsers := map[string]func([]byte) error{
			"node":    func(b []byte) error { var k key.NodePublic; return k.UnmarshalText(b) },
			"machine": func(b []byte) error { var k key.MachinePublic; return k.UnmarshalText(b) },
			"disco":   func(b []byte) error { var k key.DiscoPublic; return k.UnmarshalText(b) },
			"tl":      func(b []byte) error { var k key.TLPublic; return k.UnmarshalText(b) },
		}
		for tname, text := range texts {
			for pname, parse := range parsers {
				err := parse(text)
				if tname == pname && err != nil {
					rt.Fatalf("%s parser rejected its own text %q: %v", pname, text, err)
				}
				if tname != pname && err == nil {
					rt.Fatalf("%s parser accepted %s text %q", pname, tname, text)
				}
			}
		}
	})
}
