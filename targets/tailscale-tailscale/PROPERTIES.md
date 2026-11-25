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
`util/dnsname`, and `util/cmpver`.

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
