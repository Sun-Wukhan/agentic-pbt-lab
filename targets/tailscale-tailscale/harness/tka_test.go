package harness

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"maps"
	"reflect"
	"testing"

	"go4.org/mem"
	"pgregory.net/rapid"
	"tailscale.com/tka"
	"tailscale.com/types/key"
	"tailscale.com/types/tkatype"
)

// ---------------------------------------------------------------------------
// Wave 2 helpers: deterministic ed25519 keys + a tka.Signer implementation.
//
// Keys are derived from rapid-drawn seeds with ed25519.NewKeyFromSeed, so a
// failing check replays exactly. The signer mirrors key.TLPrivate.SignAUM:
// for a Key25519 tka.Key, the KeyID is simply the ed25519 public key.
// ---------------------------------------------------------------------------

type tkaTestKey struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	key  tka.Key
}

func tkaKeyFromSeed(seed [32]byte, votes uint) tkaTestKey {
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	return tkaTestKey{
		priv: priv,
		pub:  pub,
		key:  tka.Key{Kind: tka.Key25519, Votes: votes, Public: pub},
	}
}

func (k tkaTestKey) id() tkatype.KeyID { return tkatype.KeyID(k.pub) }

func (k tkaTestKey) idArr() [32]byte {
	var out [32]byte
	copy(out[:], k.pub)
	return out
}

type edSigner struct{ k tkaTestKey }

func (s edSigner) SignAUM(h tkatype.AUMSigHash) ([]tkatype.Signature, error) {
	return []tkatype.Signature{{
		KeyID:     s.k.id(),
		Signature: ed25519.Sign(s.k.priv, h[:]),
	}}, nil
}

func (k tkaTestKey) signer() tka.Signer { return edSigner{k} }

// DisablementKDF (argon2) costs ~120ms on this machine, so the disablement
// secret/value pairs are derived exactly once per test binary and reused
// across all rapid checks.
var (
	pbtDisablementSecrets = [][32]byte{
		sha256.Sum256([]byte("pbt-tka-disablement-secret-0")),
		sha256.Sum256([]byte("pbt-tka-disablement-secret-1")),
		sha256.Sum256([]byte("pbt-tka-disablement-secret-2")),
	}
	pbtDisablementValues = func() [][]byte {
		out := make([][]byte, len(pbtDisablementSecrets))
		for i, s := range pbtDisablementSecrets {
			out[i] = tka.DisablementKDF(s[:])
		}
		return out
	}()
)

type fataler interface {
	Fatalf(format string, args ...any)
}

// makeAuthority creates a fresh authority via tka.Create with the given
// genesis keys (signer must be one of them), returning the authority, its
// storage, and the genesis (checkpoint) AUM.
func makeAuthority(f fataler, genesis []tkaTestKey, signer tkaTestKey, stateIDs [2]uint64) (*tka.Authority, *tka.Mem, tka.AUM) {
	storage := tka.ChonkMem()
	keys := make([]tka.Key, len(genesis))
	for i, g := range genesis {
		keys[i] = g.key
	}
	state := tka.State{
		Keys:              keys,
		DisablementValues: [][]byte{pbtDisablementValues[0]},
		StateID1:          stateIDs[0],
		StateID2:          stateIDs[1],
	}
	auth, genesisAUM, err := tka.Create(storage, state, signer.signer())
	if err != nil {
		f.Fatalf("tka.Create: %v", err)
	}
	return auth, storage, genesisAUM
}

func draw32(rt *rapid.T, label string) [32]byte {
	b := rapid.SliceOfN(rapid.Byte(), 32, 32).Draw(rt, label)
	var out [32]byte
	copy(out[:], b)
	return out
}

func drawShortString(rt *rapid.T, label, alphabet string, minLen, maxLen int) string {
	n := rapid.IntRange(minLen, maxLen).Draw(rt, label+"-n")
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rapid.IntRange(0, len(alphabet)-1).Draw(rt, label+"-c")]
	}
	return string(b)
}

func drawBytes(rt *rapid.T, label string, n int) []byte {
	return rapid.SliceOfN(rapid.Byte(), n, n).Draw(rt, label)
}

// drawMeta draws a nil map or a small non-empty map. Empty-but-non-nil maps
// are never drawn: CBOR omitempty drops them on the wire, so they cannot
// round-trip (see T1).
func drawMeta(rt *rapid.T, label string) map[string]string {
	if rapid.Bool().Draw(rt, label+"-nil") {
		return nil
	}
	n := rapid.IntRange(1, 2).Draw(rt, label+"-n")
	out := make(map[string]string, n)
	for i := 0; i < n; i++ {
		k := drawShortString(rt, label+"-k", "abcde", 1, 4)
		v := drawShortString(rt, label+"-v", "xyz01", 0, 5)
		out[k+"-"+string(rune('0'+i))] = v
	}
	return out
}

func genTKAKey(rt *rapid.T, label string) tka.Key {
	return tka.Key{
		Kind:   tka.Key25519,
		Votes:  uint(rapid.IntRange(1, 4096).Draw(rt, label+"-votes")),
		Public: drawBytes(rt, label+"-pub", 32),
		Meta:   drawMeta(rt, label+"-meta"),
	}
}

func genCheckpointState(rt *rapid.T, label string) *tka.State {
	nDV := rapid.IntRange(1, 2).Draw(rt, label+"-ndv")
	dvs := make([][]byte, nDV)
	for i := range dvs {
		dv := drawBytes(rt, label+"-dv", 32)
		dv[0] = byte(i + 1) // ensure distinct values (staticValidateCheckpoint)
		dvs[i] = dv
	}
	nKeys := rapid.IntRange(1, 3).Draw(rt, label+"-nkeys")
	keys := make([]tka.Key, nKeys)
	for i := range keys {
		k := genTKAKey(rt, label+"-key")
		k.Public[0] = byte(i + 1) // ensure distinct key IDs
		keys[i] = k
	}
	return &tka.State{
		DisablementValues: dvs,
		Keys:              keys,
		StateID1:          rapid.Uint64().Draw(rt, label+"-id1"),
		StateID2:          rapid.Uint64().Draw(rt, label+"-id2"),
	}
}

// genAUM draws an AUM that satisfies AUM.StaticValidate: exactly the fields
// permitted for its kind are set (see the per-kind rules in aum.go).
func genAUM(rt *rapid.T, label string) tka.AUM {
	var a tka.AUM
	kind := rapid.IntRange(0, 4).Draw(rt, label+"-kind")
	if rapid.Bool().Draw(rt, label+"-hasprev") {
		a.PrevAUMHash = drawBytes(rt, label+"-prev", 32)
	}
	switch tka.AUMKind(kind + 1) { // skip AUMInvalid
	case tka.AUMAddKey:
		a.MessageKind = tka.AUMAddKey
		k := genTKAKey(rt, label+"-addkey")
		a.Key = &k
	case tka.AUMRemoveKey:
		a.MessageKind = tka.AUMRemoveKey
		a.KeyID = drawBytes(rt, label+"-keyid", 32)
	case tka.AUMUpdateKey:
		a.MessageKind = tka.AUMUpdateKey
		a.KeyID = drawBytes(rt, label+"-keyid", 32)
		which := rapid.IntRange(0, 2).Draw(rt, label+"-upd")
		if which == 0 || which == 2 {
			v := uint(rapid.IntRange(1, 4096).Draw(rt, label+"-votes"))
			a.Votes = &v
		}
		if which == 1 || which == 2 {
			a.Meta = drawMeta(rt, label+"-meta")
			if a.Meta == nil {
				m := map[string]string{"a": "b"}
				a.Meta = m
			}
		}
	case tka.AUMCheckpoint:
		a.MessageKind = tka.AUMCheckpoint
		a.State = genCheckpointState(rt, label+"-state")
	case tka.AUMNoOp:
		a.MessageKind = tka.AUMNoOp
	}
	nSig := rapid.IntRange(0, 2).Draw(rt, label+"-nsig")
	for i := 0; i < nSig; i++ {
		a.Signatures = append(a.Signatures, tkatype.Signature{
			KeyID:     drawBytes(rt, label+"-sigkid", 32),
			Signature: drawBytes(rt, label+"-sig", ed25519.SignatureSize),
		})
	}
	return a
}

// T1 (wave 2): Serialize -> Unserialize is the identity for generated valid
// AUMs of every kind, and re-serialization is byte-stable (so Hash is too).
func TestPBTTKAAUMRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		a := genAUM(rt, "aum")
		if err := a.StaticValidate(); err != nil {
			rt.Fatalf("generator produced invalid AUM: %v", err)
		}
		ser := a.Serialize()
		var b tka.AUM
		if err := b.Unserialize(ser); err != nil {
			rt.Fatalf("Unserialize(Serialize(aum)): %v", err)
		}
		if !reflect.DeepEqual(a, b) {
			rt.Fatalf("AUM round-trip mismatch:\n got %+v\nwant %+v", b, a)
		}
		if !bytes.Equal(b.Serialize(), ser) {
			rt.Fatalf("re-serialization differs from original serialization")
		}
		if b.Hash() != a.Hash() {
			rt.Fatalf("Hash changed across round-trip")
		}
	})
}

// T2 (wave 2): Unserialize never panics on arbitrary bytes or on mutations /
// truncations of valid AUMs; anything that decodes + static-validates also
// survives Hash/SigHash/Serialize/Weight.
func TestPBTTKAAUMUnserializeNeverPanics(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		var data []byte
		if rapid.Bool().Draw(rt, "mutate") {
			src := genAUM(rt, "aum")
			data = src.Serialize()
			switch rapid.IntRange(0, 2).Draw(rt, "mode") {
			case 0: // truncate
				data = data[:rapid.IntRange(0, len(data)).Draw(rt, "trunc")]
			case 1: // flip bytes
				n := rapid.IntRange(1, 4).Draw(rt, "nflip")
				for i := 0; i < n && len(data) > 0; i++ {
					pos := rapid.IntRange(0, len(data)-1).Draw(rt, "pos")
					data[pos] ^= byte(rapid.IntRange(1, 255).Draw(rt, "xor"))
				}
			case 2: // extend with garbage
				data = append(data, rapid.SliceOfN(rapid.Byte(), 1, 16).Draw(rt, "tail")...)
			}
		} else {
			data = rapid.SliceOfN(rapid.Byte(), 0, 256).Draw(rt, "bytes")
		}
		var a tka.AUM
		if err := a.Unserialize(data); err != nil {
			return
		}
		_ = a.Hash()
		_ = a.SigHash()
		_ = a.Serialize()
		if err := a.StaticValidate(); err == nil {
			// Weight panics on malformed signature KeyIDs by design;
			// StaticValidate guarantees they are 32 bytes.
			_ = a.Weight(tka.State{})
		}
	})
}

// ---------------------------------------------------------------------------
// T3: state-machine model over random sequences of validly-signed updates.
// ---------------------------------------------------------------------------

type modelKey struct {
	votes uint
	meta  map[string]string
}

func metaEq(a, b map[string]string) bool {
	return len(a) == len(b) && maps.Equal(a, b)
}

func checkAuthAgainstModel(rt *rapid.T, auth *tka.Authority, model map[[32]byte]modelKey) {
	keys := auth.Keys()
	if len(keys) != len(model) {
		rt.Fatalf("authority has %d keys, model has %d", len(keys), len(model))
	}
	for _, k := range keys {
		id, err := k.ID()
		if err != nil {
			rt.Fatalf("authority key ID: %v", err)
		}
		var arr [32]byte
		copy(arr[:], id)
		mk, ok := model[arr]
		if !ok {
			rt.Fatalf("authority has key %x not in model", id)
		}
		if k.Votes != mk.votes {
			rt.Fatalf("key %x votes = %d, model %d", id, k.Votes, mk.votes)
		}
		if !metaEq(k.Meta, mk.meta) {
			rt.Fatalf("key %x meta = %v, model %v", id, k.Meta, mk.meta)
		}
		if !auth.KeyTrusted(id) {
			rt.Fatalf("KeyTrusted(%x) = false for model key", id)
		}
	}
}

// T3 (wave 2): applying a random sequence of validly-signed AddKey /
// RemoveKey / SetKeyVote / SetKeyMeta updates through UpdateBuilder +
// Inform leaves the authority's head and key set exactly matching a simple
// model, and never panics.
func TestPBTTKAStateMachineModel(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		salt := rapid.Uint64().Draw(rt, "salt")
		pool := make([]tkaTestKey, 6)
		for i := range pool {
			var seedIn [40]byte
			binary.LittleEndian.PutUint64(seedIn[:8], salt)
			binary.LittleEndian.PutUint64(seedIn[8:], uint64(i))
			seed := sha256.Sum256(seedIn[:])
			votes := uint(1 + (salt+uint64(i))%3)
			pool[i] = tkaKeyFromSeed(seed, votes)
		}
		genesis := []tkaTestKey{pool[0], pool[1]}
		auth, storage, _ := makeAuthority(rt, genesis, pool[0], [2]uint64{salt, salt ^ 0x5a5a})

		model := map[[32]byte]modelKey{
			pool[0].idArr(): {votes: pool[0].key.Votes},
			pool[1].idArr(): {votes: pool[1].key.Votes},
		}
		checkAuthAgainstModel(rt, auth, model)

		nOps := rapid.IntRange(1, 20).Draw(rt, "nops")
		for op := 0; op < nOps; op++ {
			kind := rapid.IntRange(0, 6).Draw(rt, "opkind") // 0-2 add, 3-4 remove, 5 vote, 6 meta
			target := pool[rapid.IntRange(0, 5).Draw(rt, "target")]
			signer := pool[rapid.IntRange(0, 5).Draw(rt, "signer")]
			tid := target.idArr()
			_, targetIn := model[tid]
			_, signerIn := model[signer.idArr()]
			if !signerIn {
				continue // cannot produce a verifiable update
			}
			b := auth.NewUpdater(signer.signer())
			var applyToModel func()
			switch {
			case kind <= 2: // AddKey
				if targetIn {
					continue
				}
				if err := b.AddKey(target.key); err != nil {
					rt.Fatalf("builder AddKey: %v", err)
				}
				applyToModel = func() { model[tid] = modelKey{votes: target.key.Votes} }
			case kind <= 4: // RemoveKey
				if !targetIn || len(model) <= 1 {
					continue // last-key removal is rejected by aumVerify
				}
				if err := b.RemoveKey(target.id()); err != nil {
					rt.Fatalf("builder RemoveKey: %v", err)
				}
				applyToModel = func() { delete(model, tid) }
			case kind == 5: // SetKeyVote
				if !targetIn {
					continue
				}
				votes := uint(rapid.IntRange(1, 4096).Draw(rt, "votes"))
				if err := b.SetKeyVote(target.id(), votes); err != nil {
					rt.Fatalf("builder SetKeyVote: %v", err)
				}
				applyToModel = func() {
					mk := model[tid]
					mk.votes = votes
					model[tid] = mk
				}
			default: // SetKeyMeta
				if !targetIn {
					continue
				}
				meta := drawMeta(rt, "meta")
				if meta == nil {
					meta = map[string]string{}
				}
				if err := b.SetKeyMeta(target.id(), meta); err != nil {
					rt.Fatalf("builder SetKeyMeta: %v", err)
				}
				applyToModel = func() {
					mk := model[tid]
					mk.meta = meta
					model[tid] = mk
				}
			}
			updates, err := b.Finalize(storage)
			if err != nil {
				rt.Fatalf("builder Finalize: %v", err)
			}
			if len(updates) != 1 {
				rt.Fatalf("expected exactly 1 update (chain < checkpoint interval), got %d", len(updates))
			}
			if err := auth.Inform(storage, updates); err != nil {
				rt.Fatalf("Inform of builder update: %v", err)
			}
			applyToModel()
			if auth.Head() != updates[0].Hash() {
				rt.Fatalf("head = %x, want hash of applied update %x", auth.Head(), updates[0].Hash())
			}
			checkAuthAgainstModel(rt, auth, model)
			if _, removed := model[target.idArr()]; kind >= 3 && kind <= 4 && removed {
				rt.Fatalf("removed key still in model")
			}
			if kind >= 3 && kind <= 4 && auth.KeyTrusted(target.id()) {
				rt.Fatalf("KeyTrusted still true after RemoveKey")
			}
		}
	})
}

// buildOneUpdate makes a fresh authority (genesis keys A+B, signer A) and
// one valid AddKey(C) update signed by A.
func buildOneUpdate(rt *rapid.T) (genesis []tkaTestKey, signer, added tkaTestKey, update tka.AUM, ids [2]uint64) {
	salt := rapid.Uint64().Draw(rt, "salt")
	mk := func(i uint64, votes uint) tkaTestKey {
		var seedIn [40]byte
		binary.LittleEndian.PutUint64(seedIn[:8], salt)
		binary.LittleEndian.PutUint64(seedIn[8:], i)
		return tkaKeyFromSeed(sha256.Sum256(seedIn[:]), votes)
	}
	a, b, c := mk(0, 2), mk(1, 1), mk(2, 1)
	genesis = []tkaTestKey{a, b}
	ids = [2]uint64{salt, salt ^ 0xabcd}
	auth, storage, _ := makeAuthority(rt, genesis, a, ids)
	ub := auth.NewUpdater(a.signer())
	if err := ub.AddKey(c.key); err != nil {
		rt.Fatalf("builder AddKey: %v", err)
	}
	updates, err := ub.Finalize(storage)
	if err != nil {
		rt.Fatalf("builder Finalize: %v", err)
	}
	if len(updates) != 1 {
		rt.Fatalf("expected 1 update, got %d", len(updates))
	}
	return genesis, a, c, updates[0], ids
}

// T4 (wave 2): signature verification negatives — a valid update verifies,
// but a flipped signature byte, a signature from an untrusted key, stripped
// signatures, or content changed after signing are all rejected by Inform.
// Re-signing the same content with a different *trusted* key is accepted.
func TestPBTTKASignatureNegatives(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		genesis, signerA, added, valid, ids := buildOneUpdate(rt)
		trustedB := genesis[1]
		salt := rapid.Uint64().Draw(rt, "salt2")
		var seedIn [40]byte
		binary.LittleEndian.PutUint64(seedIn[:8], salt)
		// Index 77 can never collide with buildOneUpdate's indices 0,1,2,
		// even when rapid shrinks both salts to the same value.
		binary.LittleEndian.PutUint64(seedIn[8:], 77)
		untrusted := tkaKeyFromSeed(sha256.Sum256(seedIn[:]), 1)

		freshAuth := func() (*tka.Authority, *tka.Mem) {
			auth, storage, _ := makeAuthority(rt, genesis, signerA, ids)
			return auth, storage
		}
		cloneAUM := func(a tka.AUM) tka.AUM {
			out := a
			if a.Key != nil {
				k := a.Key.Clone()
				out.Key = &k
			}
			if a.Signatures != nil {
				out.Signatures = append([]tkatype.Signature(nil), a.Signatures...)
				for i := range out.Signatures {
					out.Signatures[i].KeyID = append(tkatype.KeyID(nil), a.Signatures[i].KeyID...)
					out.Signatures[i].Signature = append([]byte(nil), a.Signatures[i].Signature...)
				}
			}
			return out
		}

		// Baseline: the valid update is accepted.
		if auth, storage := freshAuth(); true {
			if err := auth.Inform(storage, []tka.AUM{valid}); err != nil {
				rt.Fatalf("baseline Inform(valid): %v", err)
			} else if !auth.KeyTrusted(added.id()) {
				rt.Fatalf("baseline: added key not trusted after Inform")
			}
		}

		// (a) Flipped signature byte -> reject.
		{
			m := cloneAUM(valid)
			pos := rapid.IntRange(0, ed25519.SignatureSize-1).Draw(rt, "sigpos")
			m.Signatures[0].Signature[pos] ^= byte(rapid.IntRange(1, 255).Draw(rt, "sigxor"))
			auth, storage := freshAuth()
			if err := auth.Inform(storage, []tka.AUM{m}); err == nil {
				rt.Fatalf("Inform accepted update with mutated signature byte")
			}
		}
		// (b) Signature from an untrusted key over the same SigHash -> reject.
		{
			m := cloneAUM(valid)
			sigs, err := untrusted.signer().SignAUM(m.SigHash())
			if err != nil {
				rt.Fatalf("untrusted SignAUM: %v", err)
			}
			m.Signatures = sigs
			auth, storage := freshAuth()
			if err := auth.Inform(storage, []tka.AUM{m}); err == nil {
				rt.Fatalf("Inform accepted update signed by untrusted key")
			}
		}
		// (c) Stripped signatures -> reject.
		{
			m := cloneAUM(valid)
			m.Signatures = nil
			auth, storage := freshAuth()
			if err := auth.Inform(storage, []tka.AUM{m}); err == nil {
				rt.Fatalf("Inform accepted unsigned update")
			}
		}
		// (d) Content changed after signing (key votes) -> reject.
		{
			m := cloneAUM(valid)
			m.Key.Votes = m.Key.Votes%4096 + 1 // different, still in valid range
			auth, storage := freshAuth()
			if err := auth.Inform(storage, []tka.AUM{m}); err == nil {
				rt.Fatalf("Inform accepted update whose content changed after signing")
			}
		}
		// (e) Control: same content re-signed by another trusted key -> accept.
		{
			m := cloneAUM(valid)
			sigs, err := trustedB.signer().SignAUM(m.SigHash())
			if err != nil {
				rt.Fatalf("trusted SignAUM: %v", err)
			}
			m.Signatures = sigs
			auth, storage := freshAuth()
			if err := auth.Inform(storage, []tka.AUM{m}); err != nil {
				rt.Fatalf("Inform rejected update re-signed by trusted key: %v", err)
			}
		}
	})
}

// T5 (wave 2): any single-byte mutation of a serialized, signed update is
// rejected — at decode, or at Inform verification — unless the mutation is
// semantically inert (decodes to the identical AUM).
func TestPBTTKASerializedMutationRejected(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		genesis, signerA, _, valid, ids := buildOneUpdate(rt)
		v := valid
		ser := v.Serialize()
		pos := rapid.IntRange(0, len(ser)-1).Draw(rt, "pos")
		mut := append([]byte(nil), ser...)
		mut[pos] ^= byte(rapid.IntRange(1, 255).Draw(rt, "xor"))

		var dec tka.AUM
		if err := dec.Unserialize(mut); err != nil {
			return // rejected at decode: fine
		}
		if reflect.DeepEqual(dec, valid) {
			return // semantically inert mutation: acceptance is correct
		}
		auth, storage, _ := makeAuthority(rt, genesis, signerA, ids)
		if err := auth.Inform(storage, []tka.AUM{dec}); err == nil {
			rt.Fatalf("Inform accepted byte-mutated update (pos %d): %+v", pos, dec)
		}
	})
}

// T6 (wave 2): checkpoint / genesis behaviour —
//   - a Create genesis (a checkpoint AUM) bootstraps a fresh authority with
//     the same head, keys and state IDs;
//   - tampered checkpoints (state edited after signing, signature flipped,
//     extra key smuggled into state, signatures stripped, or signed by a key
//     the state does not trust) are all rejected by Bootstrap;
//   - after 50 updates, UpdateBuilder.Finalize appends a checkpoint AUM, and
//     that checkpoint alone bootstraps an equivalent authority.
func TestPBTTKACheckpointBootstrap(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		salt := rapid.Uint64().Draw(rt, "salt")
		mk := func(i uint64, votes uint) tkaTestKey {
			var seedIn [40]byte
			binary.LittleEndian.PutUint64(seedIn[:8], salt)
			binary.LittleEndian.PutUint64(seedIn[8:], i)
			return tkaKeyFromSeed(sha256.Sum256(seedIn[:]), votes)
		}
		a, b, c := mk(0, 2), mk(1, 1), mk(2, 1)
		genesis := []tkaTestKey{a, b}
		ids := [2]uint64{salt, salt ^ 0x777}

		_, _, genesisAUM := makeAuthority(rt, genesis, a, ids)
		if genesisAUM.MessageKind != tka.AUMCheckpoint {
			rt.Fatalf("genesis AUM kind = %v, want checkpoint", genesisAUM.MessageKind)
		}

		// (1) Clean bootstrap.
		bootAuth, err := tka.Bootstrap(tka.ChonkMem(), genesisAUM)
		if err != nil {
			rt.Fatalf("Bootstrap(genesis): %v", err)
		}
		if bootAuth.Head() != genesisAUM.Hash() {
			rt.Fatalf("bootstrapped head = %x, want genesis hash %x", bootAuth.Head(), genesisAUM.Hash())
		}
		if id1, id2 := bootAuth.StateIDs(); id1 != ids[0] || id2 != ids[1] {
			rt.Fatalf("bootstrapped state IDs = (%d,%d), want %v", id1, id2, ids)
		}
		for _, g := range genesis {
			if !bootAuth.KeyTrusted(g.id()) {
				rt.Fatalf("bootstrapped authority does not trust genesis key %x", g.id())
			}
		}

		cloneGenesis := func() tka.AUM {
			g := genesisAUM
			st := genesisAUM.State.Clone()
			g.State = &st
			g.Signatures = append([]tkatype.Signature(nil), genesisAUM.Signatures...)
			for i := range g.Signatures {
				g.Signatures[i].Signature = append([]byte(nil), genesisAUM.Signatures[i].Signature...)
			}
			return g
		}
		expectBootFail := func(name string, g tka.AUM) {
			if _, err := tka.Bootstrap(tka.ChonkMem(), g); err == nil {
				rt.Fatalf("Bootstrap accepted tampered checkpoint: %s", name)
			}
		}
		// (2a) State edited after signing.
		g2 := cloneGenesis()
		g2.State.StateID1 ^= 1
		expectBootFail("stateID tampered", g2)
		// (2b) Signature byte flipped.
		g3 := cloneGenesis()
		g3.Signatures[0].Signature[0] ^= 0xff
		expectBootFail("signature tampered", g3)
		// (2c) Extra key smuggled into the checkpoint state.
		g4 := cloneGenesis()
		g4.State.Keys = append(g4.State.Keys, c.key)
		expectBootFail("extra key in state", g4)
		// (2d) Signatures stripped.
		g5 := cloneGenesis()
		g5.Signatures = nil
		expectBootFail("unsigned", g5)
		// (2e) Checkpoint signed by a key its own state does not trust.
		{
			state := tka.State{
				Keys:              []tka.Key{a.key},
				DisablementValues: [][]byte{pbtDisablementValues[0]},
				StateID1:          ids[0],
				StateID2:          ids[1],
			}
			forged := tka.AUM{MessageKind: tka.AUMCheckpoint, State: &state}
			sigs, err := c.signer().SignAUM(forged.SigHash())
			if err != nil {
				rt.Fatalf("forged SignAUM: %v", err)
			}
			forged.Signatures = sigs
			expectBootFail("signed by untrusted key", forged)
		}

		// (3) Long chain -> builder checkpoint -> bootstrap from it.
		// Finalize counts the pending updates toward checkpointEvery (50):
		// the walk back from head reads at most 50-len(out) stored AUMs, so
		// with 49 applied updates + 1 pending, the genesis checkpoint is
		// never reached and a checkpoint is appended to the 50th update.
		auth, storage, _ := makeAuthority(rt, []tkaTestKey{a}, a, ids)
		const total = 50
		var ckpt *tka.AUM
		var lastAdded tkaTestKey
		for i := 0; i < total; i++ {
			lastAdded = mk(uint64(100+i), 1)
			ub := auth.NewUpdater(a.signer())
			if err := ub.AddKey(lastAdded.key); err != nil {
				rt.Fatalf("chain AddKey %d: %v", i, err)
			}
			updates, err := ub.Finalize(storage)
			if err != nil {
				rt.Fatalf("chain Finalize %d: %v", i, err)
			}
			for j, u := range updates {
				if u.MessageKind == tka.AUMCheckpoint {
					if i != total-1 || j != len(updates)-1 {
						rt.Fatalf("checkpoint appeared at update %d (position %d), want only as tail of update %d", i+1, j, total)
					}
					c := u
					ckpt = &c
				}
			}
			if err := auth.Inform(storage, updates); err != nil {
				rt.Fatalf("chain Inform %d: %v", i, err)
			}
		}
		if ckpt == nil {
			rt.Fatalf("no checkpoint generated after %d updates", total)
		}
		if ckpt.State == nil {
			rt.Fatalf("checkpoint AUM has no state")
		}
		if ckpt.State.LastAUMHash != nil {
			rt.Fatalf("checkpoint state has a parent AUM hash set")
		}
		if want := 1 + total; len(ckpt.State.Keys) != want {
			rt.Fatalf("checkpoint state has %d keys, want %d", len(ckpt.State.Keys), want)
		}
		if auth.Head() != ckpt.Hash() {
			rt.Fatalf("head after checkpoint = %x, want %x", auth.Head(), ckpt.Hash())
		}
		boot2, err := tka.Bootstrap(tka.ChonkMem(), *ckpt)
		if err != nil {
			rt.Fatalf("Bootstrap(builder checkpoint): %v", err)
		}
		if len(boot2.Keys()) != 1+total {
			rt.Fatalf("bootstrapped-from-checkpoint keys = %d, want %d", len(boot2.Keys()), 1+total)
		}
		if !boot2.KeyTrusted(lastAdded.id()) || !boot2.KeyTrusted(a.id()) {
			rt.Fatalf("bootstrapped-from-checkpoint authority missing keys")
		}
	})
}

// T7 (wave 2): disablement — ValidDisablement accepts exactly the secrets
// whose KDF values are in state, and rejects other secrets and mutated
// secrets. DisablementKDF is deterministic and input-sensitive.
//
// Cost note: DisablementKDF is argon2 (~120ms here) and ValidDisablement
// runs it per call, so this property uses precomputed secret/value pairs
// and exactly three ValidDisablement calls per check.
func TestPBTTKADisablement(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		i := rapid.IntRange(0, len(pbtDisablementSecrets)-1).Draw(rt, "which")
		j := rapid.IntRange(0, len(pbtDisablementSecrets)-2).Draw(rt, "other")
		if j >= i {
			j++
		}
		salt := rapid.Uint64().Draw(rt, "salt")
		var seedIn [40]byte
		binary.LittleEndian.PutUint64(seedIn[:8], salt)
		a := tkaKeyFromSeed(sha256.Sum256(seedIn[:]), 1)

		storage := tka.ChonkMem()
		state := tka.State{
			Keys:              []tka.Key{a.key},
			DisablementValues: [][]byte{pbtDisablementValues[i]},
			StateID1:          salt,
			StateID2:          salt ^ 1,
		}
		auth, _, err := tka.Create(storage, state, a.signer())
		if err != nil {
			rt.Fatalf("Create: %v", err)
		}
		if !auth.ValidDisablement(pbtDisablementSecrets[i][:]) {
			rt.Fatalf("ValidDisablement(correct secret) = false")
		}
		if auth.ValidDisablement(pbtDisablementSecrets[j][:]) {
			rt.Fatalf("ValidDisablement(different secret) = true")
		}
		mut := append([]byte(nil), pbtDisablementSecrets[i][:]...)
		pos := rapid.IntRange(0, len(mut)-1).Draw(rt, "mutpos")
		mut[pos] ^= byte(rapid.IntRange(1, 255).Draw(rt, "mutxor"))
		if auth.ValidDisablement(mut) {
			rt.Fatalf("ValidDisablement(mutated secret) = true")
		}
		// KDF determinism is implicit above: pbtDisablementValues[i] was
		// computed by DisablementKDF at package init, and ValidDisablement
		// (which recomputes the KDF) matched it. Sensitivity: distinct
		// secrets must give distinct values (pure comparison, no KDF cost).
		if bytes.Equal(pbtDisablementValues[i], pbtDisablementValues[j]) {
			rt.Fatalf("DisablementKDF collided for distinct secrets")
		}
	})
}

// T8 (wave 2): Inform is idempotent — re-informing the same batch (or an
// already-applied prefix) succeeds and changes nothing, and an authority
// re-opened from the same storage has the same head and keys.
func TestPBTTKAInformIdempotent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		salt := rapid.Uint64().Draw(rt, "salt")
		mk := func(i uint64, votes uint) tkaTestKey {
			var seedIn [40]byte
			binary.LittleEndian.PutUint64(seedIn[:8], salt)
			binary.LittleEndian.PutUint64(seedIn[8:], i)
			return tkaKeyFromSeed(sha256.Sum256(seedIn[:]), votes)
		}
		a, b := mk(0, 2), mk(1, 1)
		auth, storage, _ := makeAuthority(rt, []tkaTestKey{a, b}, a, [2]uint64{salt, salt ^ 9})

		ub := auth.NewUpdater(a.signer())
		n := rapid.IntRange(1, 4).Draw(rt, "nadds")
		for i := 0; i < n; i++ {
			if err := ub.AddKey(mk(uint64(10+i), 1).key); err != nil {
				rt.Fatalf("builder AddKey: %v", err)
			}
		}
		updates, err := ub.Finalize(storage)
		if err != nil {
			rt.Fatalf("Finalize: %v", err)
		}
		if err := auth.Inform(storage, updates); err != nil {
			rt.Fatalf("first Inform: %v", err)
		}
		head1 := auth.Head()
		keys1 := auth.Keys()

		if err := auth.Inform(storage, updates); err != nil {
			rt.Fatalf("second Inform of same batch: %v", err)
		}
		if auth.Head() != head1 {
			rt.Fatalf("head changed after re-Inform: %x -> %x", head1, auth.Head())
		}
		if err := auth.Inform(storage, updates[:1]); err != nil {
			rt.Fatalf("Inform of already-applied prefix: %v", err)
		}
		if auth.Head() != head1 {
			rt.Fatalf("head changed after prefix re-Inform")
		}
		keys2 := auth.Keys()
		if len(keys1) != len(keys2) {
			rt.Fatalf("key count changed after re-Inform: %d -> %d", len(keys1), len(keys2))
		}
		for i := range keys1 {
			id1, _ := keys1[i].ID()
			id2, _ := keys2[i].ID()
			if !bytes.Equal(id1, id2) || keys1[i].Votes != keys2[i].Votes {
				rt.Fatalf("key %d changed after re-Inform", i)
			}
		}

		reopened, err := tka.Open(storage)
		if err != nil {
			rt.Fatalf("Open after Inform: %v", err)
		}
		if reopened.Head() != head1 {
			rt.Fatalf("reopened head = %x, want %x", reopened.Head(), head1)
		}
		if len(reopened.Keys()) != len(keys1) {
			rt.Fatalf("reopened key count = %d, want %d", len(reopened.Keys()), len(keys1))
		}
	})
}

// T9 (wave 2): NodeKeyAuthorized — a SigDirect NodeKeySignature signed by a
// trusted TKA key authorizes its node key; the same signature with a flipped
// signature byte, presented for a different node key, or signed by an
// untrusted key is rejected.
func TestPBTTKANodeKeyAuthorized(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		salt := rapid.Uint64().Draw(rt, "salt")
		mk := func(i uint64, votes uint) tkaTestKey {
			var seedIn [40]byte
			binary.LittleEndian.PutUint64(seedIn[:8], salt)
			binary.LittleEndian.PutUint64(seedIn[8:], i)
			return tkaKeyFromSeed(sha256.Sum256(seedIn[:]), votes)
		}
		a, untrusted := mk(0, 2), mk(1, 1)
		auth, _, _ := makeAuthority(rt, []tkaTestKey{a}, a, [2]uint64{salt, salt ^ 3})

		nodeRaw := draw32(rt, "node")
		if nodeRaw == ([32]byte{}) {
			nodeRaw[0] = 1
		}
		nodePriv := key.NodePrivateFromRaw32(mem.B(nodeRaw[:]))
		nodePub := nodePriv.Public()
		nodePubBytes, err := nodePub.MarshalBinary()
		if err != nil {
			rt.Fatalf("nodePub MarshalBinary: %v", err)
		}

		mkSig := func(signer tkaTestKey, pubBytes []byte) tka.NodeKeySignature {
			s := tka.NodeKeySignature{
				SigKind: tka.SigDirect,
				KeyID:   signer.id(),
				Pubkey:  pubBytes,
			}
			h := s.SigHash()
			s.Signature = ed25519.Sign(signer.priv, h[:])
			return s
		}

		good := mkSig(a, nodePubBytes)
		if err := auth.NodeKeyAuthorized(nodePub, good.Serialize()); err != nil {
			rt.Fatalf("NodeKeyAuthorized(valid direct sig): %v", err)
		}
		// Flipped signature byte.
		bad := mkSig(a, nodePubBytes)
		bad.Signature[rapid.IntRange(0, ed25519.SignatureSize-1).Draw(rt, "pos")] ^= 0x40
		if err := auth.NodeKeyAuthorized(nodePub, bad.Serialize()); err == nil {
			rt.Fatalf("NodeKeyAuthorized accepted flipped signature")
		}
		// Presented for a different node key.
		otherRaw := draw32(rt, "othernode")
		if otherRaw == ([32]byte{}) || otherRaw == nodeRaw {
			otherRaw[0] ^= 0x80
		}
		otherPub := key.NodePrivateFromRaw32(mem.B(otherRaw[:])).Public()
		if err := auth.NodeKeyAuthorized(otherPub, good.Serialize()); err == nil {
			rt.Fatalf("NodeKeyAuthorized accepted sig for a different node key")
		}
		// Signed by an untrusted key.
		forged := mkSig(untrusted, nodePubBytes)
		if err := auth.NodeKeyAuthorized(nodePub, forged.Serialize()); err == nil {
			rt.Fatalf("NodeKeyAuthorized accepted sig from untrusted key")
		}
	})
}

// T10 (wave 2): NodeKeySignature Serialize -> Unserialize round-trips for
// generated shapes (direct, credential, rotation nesting), and Unserialize
// never panics on arbitrary bytes.
func TestPBTTKANKSRoundTripNoPanic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		genNKS := func(depth int) *tka.NodeKeySignature {
			kind := rapid.IntRange(0, 1).Draw(rt, "leafkind")
			leaf := &tka.NodeKeySignature{
				KeyID: drawBytes(rt, "kid", 32),
			}
			if kind == 0 {
				leaf.SigKind = tka.SigDirect
				leaf.Pubkey = drawBytes(rt, "pub", 34)
			} else {
				leaf.SigKind = tka.SigCredential
				leaf.WrappingPubkey = drawBytes(rt, "wrap", 32)
			}
			leaf.Signature = drawBytes(rt, "sig", ed25519.SignatureSize)
			out := leaf
			for i := 0; i < depth; i++ {
				out = &tka.NodeKeySignature{
					SigKind:   tka.SigRotation,
					Pubkey:    drawBytes(rt, "rpub", 34),
					Signature: drawBytes(rt, "rsig", ed25519.SignatureSize),
					Nested:    out,
				}
				if rapid.Bool().Draw(rt, "rwrap") {
					out.WrappingPubkey = drawBytes(rt, "rwrapk", 32)
				}
			}
			return out
		}
		s := genNKS(rapid.IntRange(0, 2).Draw(rt, "depth"))
		ser := s.Serialize()
		var back tka.NodeKeySignature
		if err := back.Unserialize(ser); err != nil {
			rt.Fatalf("NKS Unserialize(Serialize): %v", err)
		}
		if !reflect.DeepEqual(*s, back) {
			rt.Fatalf("NKS round-trip mismatch:\n got %+v\nwant %+v", back, *s)
		}
		if !bytes.Equal(back.Serialize(), ser) {
			rt.Fatalf("NKS re-serialization differs")
		}

		garbage := rapid.SliceOfN(rapid.Byte(), 0, 256).Draw(rt, "garbage")
		var g tka.NodeKeySignature
		if err := g.Unserialize(garbage); err == nil {
			_ = g.Serialize()
			_ = g.SigHash()
		}
	})
}

// T11 (wave 2): AUMHash text (base32) round-trips, and UnmarshalText rejects
// wrong-length input without panicking.
func TestPBTTKAAUMHashTextRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		raw := draw32(rt, "hash")
		var h tka.AUMHash
		copy(h[:], raw[:])
		text, err := h.MarshalText()
		if err != nil {
			rt.Fatalf("AUMHash MarshalText: %v", err)
		}
		var back tka.AUMHash
		if err := back.UnmarshalText(text); err != nil {
			rt.Fatalf("AUMHash UnmarshalText: %v", err)
		}
		if back != h {
			rt.Fatalf("AUMHash text round-trip mismatch")
		}
		if h.String() != string(text) {
			rt.Fatalf("AUMHash String() != MarshalText")
		}
		garbage := rapid.SliceOfN(rapid.Byte(), 0, 64).Draw(rt, "garbage")
		var g tka.AUMHash
		_ = g.UnmarshalText(garbage) // must not panic; error is fine
	})
}
