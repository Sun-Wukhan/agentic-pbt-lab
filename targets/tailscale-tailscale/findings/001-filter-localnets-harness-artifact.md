# Finding: filter model mismatch when localNets is nil

- Target / property: tailscale/tailscale @ 445df50dcf76b6172dc4688c2097c47878f41641, property F1 (`TestPBTFilterCheckVsModel`)
- Minimal counterexample: one rule `{TCP, src 10.0.0.0/8, dst 10.0.0.0/8, ports 0-0}`; query `Check(10.0.0.0, 10.0.0.0, 0, TCP)` returned `Drop`; the naive model said Accept. Rapid seed 17423811707361107215 (checks=100 run, failed after 6 tests).
- Observed vs expected: observed Drop, model expected Accept.
- Verdict: harness artifact (wrong harness setup, not an upstream bug)
- Reasoning: `wgengine/filter/filter.go` `runIn4`/`runIn6` first check whether the packet destination is a local address: `if !f.local4(q.Dst.Addr()) { return noVerdict, "destination not allowed" }`. The harness initially passed `localNets=nil` to `filter.New`, which installs `ipset.FalseContainsIPFunc()` — so *every* inbound packet was dropped before the match rules were consulted, whatever the rules said. In production the filter is always constructed with the node's real local addresses/routes. Fix was harness-side: pass an `IPSet` covering the generated destination space (10.0.0.0/8). After the fix, F1 passes at checks=100 and checks=2000.
- Upstream issue: none (nothing filed, per lab rules)
- Fix PR: none
