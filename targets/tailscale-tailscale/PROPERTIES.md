# Properties — tailscale/tailscale

Target commit: `445df50dcf76b6172dc4688c2097c47878f41641` (main, cloned 2026-11-25)
Framework: Go, [pgregory.net/rapid](https://pgregory.net/rapid) v1.2.0
Harness layout: a separate Go module under `harness/` that depends on the
target through a replace directive pointing at the pinned clone in
`./tailscale` — the clone's own files are never modified. `run.sh` clones
the target at the pinned commit if needed, verifies the pin, and runs
every `TestPBT*` property.

Surfaces, in priority order: `net/art` (routing table vs a brute-force
longest-prefix-match model), `net/packet` + `wgengine/filter` (packet
codecs and ACL match decisions), `disco` + `derp` (message/frame codecs),
`util/dnsname`, and `util/cmpver`. Wave 2 (below) adds `tka` (tailnet
key authority: AUM codecs, the signed-update state machine, checkpoints,
disablement, node-key authorization) and `types/key` (key text/binary
round-trips and parser robustness).

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| A1 | With a single route inserted, `Get(addr)` returns its value iff the (masked) route contains addr | `net/art/table.go` `Get`/`Insert` doc comments | `TestPBTArtSingleInsertGet` |
| A2 | After any sequence of inserts/deletes, `Get` agrees with a brute-force longest-prefix-match model (most bits wins, last write wins) | `net/art/table.go` — package doc (ART routing table) + `Get`/`Insert`/`Delete` semantics | `TestPBTArtModelVsSequence` |
| A3 | Inserting the same prefix twice overwrites: last write wins | `net/art/table.go` `Insert` doc: "If pfx is already present in the table, its value is set to val" | `TestPBTArtOverwriteLastWins` |
| A4 | `Insert` masks its argument, so an unmasked prefix behaves exactly like its masked form, including for `Delete` | `net/art/table.go` `Insert` body comment: algorithms require normalized prefixes, "so do it upfront" (`pfx = pfx.Masked()`) | `TestPBTArtInsertMasksPrefix` |
| A5 | Family isolation: IPv4 routes never answer IPv6 lookups and vice versa | `net/art/table.go` `Table` struct (separate v4/v6 stride tables) + `tableForAddr` | `TestPBTArtFamilyIsolation` |
| P1 | `packet.Parsed.Decode` never panics on arbitrary bytes | `net/packet/packet.go` `Decode` — decodes untrusted wire packets in the TUN path | `TestPBTPacketDecodeNeverPanics` |
| P2 | `Decode` never panics on any truncation of a valid packet | `net/packet/packet.go` `Decode` (same untrusted-input reasoning) | `TestPBTPacketDecodeTruncationsNeverPanic` |
| P3 | `Generate(UDP4Header, payload)` → `Decode` round-trips version, protocol, src/dst addr:port and payload | `net/packet/header.go` `Generate`/`Header` contract + `udp4.go`/`ip4.go` `Marshal` | `TestPBTPacketUDP4RoundTrip` |
| P4 | `Generate(IP4Header, payload)` with a non-TCP/UDP/ICMP protocol → `Decode` round-trips version, protocol and addresses | `net/packet/ip4.go` `Marshal` + `packet.go` `Parsed` field docs | `TestPBTPacketIP4RoundTrip` |
| P5 | `Decode` is deterministic: same bytes decode to identical exported fields twice | `net/packet/packet.go` `Decode` (pure function of the buffer) | `TestPBTPacketDecodeDeterministic` |
| F1 | `Filter.Check` (TCP/UDP, IPv4 unicast, destination in localNets) agrees with a naive reference model over the same rule set: accept iff some rule matches proto, src prefix, dst prefix and port range | `wgengine/filter/filter.go` `Check` doc + `filtertype.Match` doc ("matches packets from any IP address in Srcs to any ip:port in Dsts"); local-destination precondition from `runIn4`/`runIn6` ("destination not allowed") | `TestPBTFilterCheckVsModel` |
| F2 | With no rules, `Check` drops everything | `wgengine/filter/filter.go` — default-deny ACL semantics | `TestPBTFilterEmptyDropsAll` |
| F3 | Mismatched address families are dropped, even with allow-all rules in both families | `wgengine/filter/filter.go` `Check` body comment: "Mismatched address families, no filters will match" | `TestPBTFilterMismatchedFamiliesDrop` |
| D1 | `Ping` AppendMarshal → `disco.Parse` round-trips TxID and NodeKey (zero and non-zero keys) | `disco/disco.go` `Ping`/`parsePing` (parser is "deliberately lax on longer-than-expected messages"; zero keys are omitted on the wire and must come back zero) | `TestPBTDiscoPingRoundTrip` |
| D2 | `Pong` AppendMarshal → `Parse` round-trips TxID and Src | `disco/disco.go` `Pong`/`parsePong` | `TestPBTDiscoPongRoundTrip` |
| D3 | `CallMeMaybe` AppendMarshal → `Parse` round-trips the endpoint list | `disco/disco.go` `CallMeMaybe`/`parseCallMeMaybe` (wire form Unmaps addresses, so generators use unmapped forms) | `TestPBTDiscoCallMeMaybeRoundTrip` |
| D4 | `disco.Parse` never panics on arbitrary bytes or any truncation of a valid message | `disco/disco.go` `Parse` — parses decrypted-but-peer-controlled payloads | `TestPBTDiscoParseNeverPanics` |
| R1 | A frame header encoded as (type byte + big-endian uint32 length) is read back exactly by `derp.ReadFrameHeader` | `derp/derp.go` `ReadFrameHeader` + `FrameHeaderLen` wire format | `TestPBTDerpFrameHeaderRoundTrip` |
| R2 | `derp.ReadFrameHeader` never panics on arbitrary/truncated input and errors when fewer than `FrameHeaderLen` bytes are available | `derp/derp.go` `ReadFrameHeader` (Peek-based; short reads error) | `TestPBTDerpFrameHeaderNeverPanics` |
| N1 | A successful `ToFQDN` is a fixed point: trailing dot present, re-parsing returns it unchanged, `NumLabels` matches the input label count; rejection is only acceptable when a label exceeds 63 bytes or the name exceeds the length limit | `util/dnsname/dnsname.go` `ToFQDN` doc + RFC 1035 limits in its constants | `TestPBTDnsnameToFQDNFixedPoint` |
| N2 | `ToFQDN` rejects whitespace/control characters, empty labels, labels over 63 bytes, and names over 254 bytes | `util/dnsname/dnsname.go` `ToFQDN` body comments (control-server injection rationale; `maxLabelLength`/`maxNameLength`) | `TestPBTDnsnameToFQDNRejects` |
| N3 | `SanitizeLabel` output is empty or accepted by `ValidLabel`, is lowercase, and sanitizing is idempotent | `util/dnsname/dnsname.go` `SanitizeLabel` doc ("turns it into a valid name label according to RFC 1035") | `TestPBTDnsnameSanitizeLabel` |
| N4 | `FQDN.Parent` strips exactly the first label (empty for single-label names) and `Contains` agrees with the label-suffix model (self and parent contain; child does not contain parent) | `util/dnsname/dnsname.go` `Parent`/`Contains` doc comments | `TestPBTDnsnameParentContains` |
| C1 | `cmpver.Compare` is antisymmetric and reflexive; `Less`/`LessEq` agree with it | `util/cmpver/version.go` `Compare`/`Less`/`LessEq` doc comments | `TestPBTCmpverAntisymmetry` |
| C2 | `Compare` induces a total order: transitivity over generated triples, including equality | `util/cmpver/version.go` package doc (pairwise field comparison determines ordering) | `TestPBTCmpverTransitivity` |
| C3 | For dotted-numeric versions, `Compare` agrees with an independent model: component-wise big-integer comparison, and if all shared components are equal the version with more components is greater | `util/cmpver/version.go` package doc + `Less` doc note: "12" is less than "12.0" | `TestPBTCmpverDottedNumericModel` |

## Wave 2 — tka (tailnet key authority) + types/key

Same pinned commit, same harness module. The tka surface is exercised
entirely through exported API: `tka.Create` / `Bootstrap` / `Open`,
`Authority.Inform`, `UpdateBuilder` (`Authority.NewUpdater`), `tka.Mem`
(in-memory Chonk), and a harness `tka.Signer` that mirrors
`key.TLPrivate.SignAUM` (for a Key25519 key, the KeyID is the ed25519
public key). Test keys are derived deterministically from rapid-drawn
seeds via `ed25519.NewKeyFromSeed`. The package's own test helpers
(`chaintest_test.go` etc.) are `_test.go` files inside package tka and
cannot be imported, so the harness re-implements the small pieces it
needs. `Authority.Keys()` / `KeyTrusted()` / `Head()` / `StateIDs()` /
`ValidDisablement()` serve as the model oracle.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| T1 | `AUM.Serialize` → `Unserialize` is the identity for generated valid AUMs of every kind (AddKey/RemoveKey/UpdateKey/Checkpoint/NoOp, with/without parent, 0–2 signatures); re-serialization is byte-stable, so `Hash` is stable | `tka/aum.go` `Serialize`/`Unserialize` (CBOR CTAP2, deterministic because signatures are over the serialization) + per-kind field rules in `StaticValidate` | `TestPBTTKAAUMRoundTrip` |
| T2 | `AUM.Unserialize` never panics on arbitrary bytes or on truncations/mutations/extensions of valid AUMs; decoded+statically-valid AUMs survive `Hash`/`SigHash`/`Serialize`/`Weight` | `tka/aum.go` `Unserialize` + `tka/tka.go` strict CBOR decoder settings (AUMs are parsed from untrusted peers during sync); complements upstream `FuzzAUMUnserializeValidate` | `TestPBTTKAAUMUnserializeNeverPanics` |
| T3 | Over random sequences of validly-signed AddKey/RemoveKey/SetKeyVote/SetKeyMeta updates applied via `UpdateBuilder` + `Inform`, the authority never panics, `Head()` is the last applied AUM's hash, and the key set (IDs, votes, meta) exactly matches a simple model; removed keys are no longer trusted | `tka/state.go` `applyVerifiedAUM` per-kind semantics; `tka/builder.go` `UpdateBuilder`; `tka/tka.go` `aumVerify` (incl. "cannot remove the last key") and `Inform` | `TestPBTTKAStateMachineModel` |
| T4 | Signature verification negatives: a valid builder update is accepted, but the same update with a flipped signature byte, a signature from an untrusted key, stripped signatures, or content changed after signing is rejected by `Inform`; re-signing with a different *trusted* key is accepted | `tka/tka.go` `aumVerify` + `tka/verify.go` `signatureVerify` (signatures verified against keys in the pre-update state) | `TestPBTTKASignatureNegatives` |
| T5 | Any single-byte mutation of a serialized signed update is rejected (at decode or at `Inform`), unless it decodes to the identical AUM | `tka/aum.go` `SigHash` (signature covers all fields except Signatures) + `aumVerify` | `TestPBTTKASerializedMutationRejected` |
| T6 | Checkpoint/genesis behaviour: a `Create` genesis (checkpoint AUM) bootstraps a fresh authority with identical head/keys/state IDs; tampered checkpoints (state edited after signing, flipped signature, extra key smuggled into state, stripped signatures, signed by a key the state does not trust) are rejected by `Bootstrap`; `Finalize` appends a checkpoint to the update that brings the count since the last checkpoint to 50 (`checkpointEvery`), and that checkpoint alone bootstraps an equivalent authority | `tka/tka.go` `Create`/`Bootstrap` (bootstrap AUMs must be checkpoints, verified against their own state); `tka/builder.go` `Finalize`/`generateCheckpoint` + `checkpointEvery = 50` (pending updates count toward the interval — see findings/003) | `TestPBTTKACheckpointBootstrap` |
| T7 | `Authority.ValidDisablement` accepts exactly the secrets whose `DisablementKDF` values are in state, and rejects other secrets and one-byte mutations of the right secret; KDF values for distinct secrets differ | `tka/state.go` `DisablementKDF`/`checkDisablement` docs; `tka/tka.go` `ValidDisablement` | `TestPBTTKADisablement` |
| T8 | `Inform` is idempotent: re-informing the same batch or an already-applied prefix succeeds and leaves head and keys unchanged; `Open` on the same storage recovers the same head and key count | `tka/tka.go` `InformIdempotent` doc ("If the set of updates is a chain building from the current head...") + `Open`/`computeActiveChain` | `TestPBTTKAInformIdempotent` |
| T9 | `Authority.NodeKeyAuthorized` accepts a `SigDirect` `NodeKeySignature` signed by a trusted key for its node key, and rejects the same signature with a flipped byte, presented for a different node key, or signed by an untrusted key | `tka/sig.go` `NodeKeySignature`/`verifySignature` (SigDirect: signature over SigHash by the TKA key named by KeyID, Pubkey must equal the node key's binary form); `tka/tka.go` `NodeKeyAuthorized` | `TestPBTTKANodeKeyAuthorized` |
| T10 | `NodeKeySignature.Serialize` → `Unserialize` round-trips generated shapes (direct, credential, rotation nested 0–2 deep); `Unserialize` never panics on arbitrary bytes | `tka/sig.go` `Serialize`/`Unserialize`; complements upstream `FuzzNodeKeySignatureUnserialize` | `TestPBTTKANKSRoundTripNoPanic` |
| T11 | `AUMHash` text form (base32, no padding) round-trips via `MarshalText`/`UnmarshalText` and agrees with `String()`; garbage text is rejected or ignored without panicking | `tka/aum.go` `AUMHash.MarshalText`/`UnmarshalText`/`String` | `TestPBTTKAAUMHashTextRoundTrip` |
| K1 | `types/key` text round-trips: NodePrivate/NodePublic, MachinePrivate/MachinePublic, DiscoPublic and TLPrivate/TLPublic (including the `tlpub:` CLI form) survive MarshalText → UnmarshalText unchanged; `Public()` is deterministic and survives the private key's own round-trip; NodePublic MarshalBinary round-trips; Disco `Shared()` is symmetric (X25519 agreement); TL KeyIDs agree across `TLPrivate.KeyID`, `TLPublic.KeyID` and `tka.Key.ID` | `types/key/node.go`, `machine.go`, `disco.go`, `tl.go` Marshal/Unmarshal implementations and prefix constants; `tl.go` `KeyID` comment (KeyID is the ed25519 public) | `TestPBTKeyTextRoundTrips` |
| K2 | Key parsers never panic on arbitrary or mutated input; anything a parser accepts re-marshals to a stable form that re-parses to an equal key (canonical acceptance) | `types/key/util.go` `parseHex` (prefix + fixed-length hex contract) | `TestPBTKeyParsersNeverPanic` |
| K3 | Public-key text parsers reject each other's prefixes (`nodekey:` ≠ `mkey:` ≠ `discokey:` ≠ `nlpub:`), while each accepts its own | `types/key` per-type hex prefix constants (`nodePublicHexPrefix`, `machinePublicHexPrefix`, `discoPublicHexPrefix`, `tlPublicHexPrefix`) | `TestPBTKeyCrossPrefixRejection` |

### Wave 2 limits (intractable in a standalone harness)

- **Fork resolution** (`pickNextAUM`/`advanceByPrimary` weight-based
  branch choice) is not modelled: constructing meaningful forks needs
  competing signed chains plus sync offers, and the "correct" winner
  depends on signature weights computed over states the harness would
  have to reimplement. Out of scope for this wave.
- **Sync** (`SyncOffer`/`MissingAUMs`) and **compaction**
  (`Authority.Compact`, `MakeRetroactiveRevocation`) need multi-authority
  storage topologies; not covered. `Inform`/`Open` exercise the shared
  chain-recomputation code paths they build on.
- **SigRotation / SigCredential NKS chains** are only round-tripped
  (T10), not verified end-to-end: `NodeKeyAuthorized` verification of
  rotations needs wrapped auth keys via `SignByCredential` and node-key
  rotation flows that live in `ipn/ipnlocal`, outside a pure harness.
- **DERP-style storage backends** for Chonk (the filesystem `ChonkDir`)
  are not exercised; `tka.Mem` covers the `Chonk` interface semantics
  the state machine depends on.
- `DisablementKDF` is argon2 (~120ms/call on this box) and
  `ValidDisablement` runs it per call, so T7 uses secret/value pairs
  precomputed once per test binary and exactly three `ValidDisablement`
  calls per check. T7 dominates the checks=2000 runtime.

## Notes

- ART generators draw addresses from small pools (one IPv4 /8-ish space
  and three IPv6 clusters) so overlaps, nesting and collisions are
  common; prefixes are deliberately not always masked (A4).
- Filter scope is deliberately narrow: `Check` only, TCP/UDP, IPv4
  unicast in 10.0.0.0/8, with localNets covering that space (see
  findings/001 for why localNets matters). ICMP (port-insensitive,
  stateful echo handling), TSMP, flow-tracked UDP return traffic and
  capability-based matches are out of model scope.
- One suspicious line was noticed while reading `Check` but is *not*
  counted as a finding: the family-mismatch guard reads
  `(srcIP.Is6() && srcIP.Is4())` for its second clause, which can never
  be true (presumably `dstIP.Is4()` was meant). F3 still passes because
  the v6-source/v4-destination case is dropped downstream (the v4
  destination is not in the v6 local set in `runIn6`). Recorded here
  for a human to judge; the observable `Check` contract holds.
- DERP's rich frame types are only reachable through client/server
  connections, not pure exported marshal/parse functions, so the DERP
  surface is limited to the exported frame-header reader (R1/R2).
- Results: checks=100 — all 26 properties pass (one harness-artifact
  counterexample during bring-up, findings/001). checks=2000 long run —
  see `results/checks-2000.log` and the README scoreboard.
- Wave 2 results: 14 new properties (T1–T11, K1–K3; 40 total). All pass
  at checks=100 and checks=2000 — see `results/checks-100-wave2.log` and
  `results/checks-2000-wave2.log`. Two counterexamples during bring-up,
  both harness-side: a generator key collision that made the "untrusted"
  key secretly trusted (findings/002) and an off-by-one in the harness's
  model of when `Finalize` appends a checkpoint (findings/003). No
  confirmed upstream bugs.
