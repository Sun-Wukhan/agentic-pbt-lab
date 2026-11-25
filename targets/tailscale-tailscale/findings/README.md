# Findings — tailscale/tailscale

Runs at checks=100 and checks=2000 (26 properties, all passing at both
levels after bring-up fixes). Logs: `../results/checks-100.log`,
`../results/checks-2000.log`.

| # | Finding | Verdict |
|---|---------|---------|
| 001 | Filter model mismatch when the harness passed `localNets=nil` (every inbound packet dropped as "destination not allowed" before rules were consulted) | harness artifact — fixed harness-side, no upstream bug |

No confirmed upstream bugs were found in this hunt. One code smell
noticed by reading (not a counterexample): the family-mismatch guard in
`wgengine/filter.Filter.Check` contains a never-true clause
`(srcIP.Is6() && srcIP.Is4())`; the observable contract (mismatched
families are dropped) still holds via downstream checks. See the Notes
section of `../PROPERTIES.md`.
