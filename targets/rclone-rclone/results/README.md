# Run record — rclone/rclone @ `1053463755dea408a4abd42bf2ce917a3884a62a`

Date: 2026-10-28 · Framework: pgregory.net/rapid v1.2.0 · Go 1.25.x
Environment note: this sandbox's `/tmp` is a 512 MB tmpfs, so all runs used
`GOCACHE=~/workspace/.gocache`, `GOTMPDIR=~/workspace/.gotmp`,
`RCLONE_CONFIG=/notfound`. The VM was under heavy parallel load throughout;
several compiles took minutes. Wall-clock times in logs are not meaningful.

## Logs

| File | Contents |
|------|----------|
| `run-default-fs.log` | fs + fspath + filter properties, default 100 checks. Contains the shrunk counterexamples for F3–F7 and the original L3 harness failure (Finding 004). |
| `run-l3-fixed.log` | L3 rerun after the harness fix (`NewFilter(nil)`): PASS 100. |
| `run-default-crypt.log` | crypt C1–C3, default 100 checks: all PASS. |
| `run-default-chunker.log` | chunker H1–H2, default 100 checks: all PASS (see isolation note). |
| `run-long-fs.log` | F1, F2 at `-rapid.checks=2000`: PASS 2000/2000. |
| `run-long-fspath-filter.log` | S1, S2, L1, L2, L3 at 2000 checks: all PASS. |
| `run-long-crypt.log` | C1–C3 at 2000 checks. |
| `run-long-chunker.log` | H1, H2 at 2000 checks (isolation note applies). |
| `repro-counterexamples.log` | Plain-Go triage reproducers for Findings 001–003 (source: `../findings/reproducers_test.go.txt`). |
| `run-default-checks-part1.log` | Aborted first attempt (killed during module downloads; no tests ran). Kept for completeness. |

## Chunker compile-isolation note

`go test ./backend/chunker/` links upstream's own test files
(`chunker_test.go`, `chunker_internal_test.go`), which import the fstest
framework and thereby all 70+ backends; that link did not complete in
>30 minutes in this environment. The chunker properties were therefore run
with those two upstream test files temporarily renamed inside the scratch
clone (restored afterwards; the clone's non-test code was untouched, and
the properties exercise only non-test code: `unmarshalSimpleJSON` /
`marshalSimpleJSON`). `run.sh` performs the canonical full-package run;
on an unloaded machine it needs no isolation.

## Scoreboard numbers

- Properties encoded: 17 (F1–F7, S1–S2, L1–L3, C1–C3, H1–H2)
- Passed at 100 checks and again at 2,000 checks: 12
- Failed: 5 properties (F3, F4, F5, F6, F7) → Findings 001–003
  (2 confirmed bugs, of which 1 is reportable by us; 1 wrong property).
  Finding 004 records a sixth, initial failure (L3) that was a harness
  artifact; L3 passes at both check counts after the harness fix.
- Confirmed bugs: 2 (Finding 001 — already claimed upstream, PR #10154;
  Finding 002 — reportable, fix in progress separately)
