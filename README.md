# Agentic Property-Based Testing Lab

Finding real bugs in open-source software with property-based testing (PBT),
following the method of Maaz, DeVoe, Hatfield-Dodds & Carlini,
"Agentic Property-Based Testing: Finding Bugs Across the Python Ecosystem"
(Anthropic / MATS, 2025 — https://www.anthropic.com/research/property-based-testing,
paper https://arxiv.org/abs/2510.09907, reference code https://github.com/mmaaz-git/agentic-pbt).

## Method

For each target project:
1. **Infer properties** — read the code and docs and write down invariants that
   should always hold (round-trips, idempotence, ordering, no-crash,
   conservation, model equivalence, metamorphic relations).
2. **Synthesize tests** — express each property as an executable PBT
   (Go: pgregory.net/rapid + native fuzzing; Python: Hypothesis;
   TypeScript: fast-check).
3. **Run at scale** — the framework searches for counterexamples and shrinks
   any failure to a minimal repro.
4. **Reflect and triage** — every counterexample is reviewed by a human-grade
   pass: is it a true bug, a wrong property, or intended behaviour? Verdicts
   and reasoning are recorded in the target's `findings/` — including the
   false positives, in the open.
5. **Rank and report** — only high-scoring confirmed bugs are reported
   upstream, with the minimized repro and, where we can, a patch. No
   mass-filing: the paper found 56% of raw reports valid; our rubric exists
   to report only from the top of the ranking, where they found 86% valid.

## Layout

- `targets/<owner>-<repo>/PROPERTIES.md` — the inferred invariants, each with
  where it came from (file/doc) and the test that encodes it.
- `targets/<owner>-<repo>/` — the harness code and a `run.sh` so anyone can
  reproduce our runs exactly.
- `targets/<owner>-<repo>/findings/` — one file per counterexample: the
  property, the minimal failing input, our verdict, and links to any
  upstream issue/PR.

## Running the labs

Every target is self-contained and reproducible: `targets/<slug>/run.sh`
is the whole recipe. `./run-all.sh` at the repo root runs every built lab
and writes one log per target under `results/`. The target list and waves
live in `targets.yaml`.

## Scoreboard

| Target | Properties | Runs | Counterexamples | Confirmed bugs | Reported | Fixed |
|--------|-----------|------|-----------------|----------------|----------|-------|
| rclone/rclone | 17 | 100 + 2,000 rapid checks | 5 property failures (3 findings) + 1 harness artifact | 2 (1 already claimed upstream, PR #10154) | 1 by us (#10205) | fix PR open (#10206) |
| pytorch/pytorch | 27 | 200 + 2,000 Hypothesis examples | 1 probe failure by design + 4 further counterexamples during bring-up/long run (5 findings) | 0 — every counterexample triaged to a wrong property, a documented divergence (negative-step slicing), or our own harness | 0 (nothing to report) | n/a |
| kubernetes/apimachinery | 24 + 18 (Quantity wave 2) | 100 + 2,000 rapid checks (both waves) | 7 findings (4 wave 1; 3 Quantity wave 2) | 1 (`metav1.ParseToLabelSelector` drops conflicting equalities; the Quantity exponent-wrap candidate was triaged as deliberate upstream behaviour, restored for etcd compatibility) | 0 (held; fix prepared locally, issue draft in target dir) | — |
| Masterminds/semver | 22 | 100 + 2,000 rapid checks | 3 findings (2 in our harness/generators, 1 in upstream code) | 1 (prerelease numeric identifiers >uint64 compared lexically, `comparePrePart`) | 1 by us ([#326](https://github.com/Masterminds/semver/pull/326) — duplicates earlier [#323](https://github.com/Masterminds/semver/pull/323), which carries the same fix; our pre-submission search missed it, recorded here openly) | — (fix carried by #323) |
| numpy/numpy (2.5.4) | 25 | 200 + 2,000 Hypothesis examples | 1 property failure (P3, promotion associativity) | 0 — the failure is NumPy's documented "rarely associative" promotion (wrong property) | 0 | — |
| pypa/packaging (26.4.dev0 @ 713d0d78) | 36 | 500 + 2,000 Hypothesis examples | 2 (Requirement round-trip on a `===`-combined specifier; manual validator probe) | 2 (`Requirement` rejects PEP 508-valid `x===0,>= 0` — the tokenizer's `===` token swallows separators; `from_parts` validators accept `bool` parts and mint unparseable versions, low severity) | 2 by us (issues [#1500](https://github.com/pypa/packaging/issues/1500), [#1499](https://github.com/pypa/packaging/issues/1499); PRs [#1502](https://github.com/pypa/packaging/pull/1502), [#1501](https://github.com/pypa/packaging/pull/1501)) | PRs open |
| encode/httpx (main @ b5addb6) | 27 | 500 + 2,000 Hypothesis examples | 9 property failures (7 findings) | 6 (scheme-case port elision, IDNA `.host` decode gated on first label, `.host` leaking raw idna exceptions, dot-segment trailing-slash loss vs RFC 3986 §5.2.4, leading-`:` scheme-delimiter bypass, `Headers.__len__` counting duplicates) | 5 "Potential Issue" discussions by us ([#3817](https://github.com/encode/httpx/discussions/3817), [#3818](https://github.com/encode/httpx/discussions/3818), [#3819](https://github.com/encode/httpx/discussions/3819), [#3820](https://github.com/encode/httpx/discussions/3820), [#3821](https://github.com/encode/httpx/discussions/3821)) — upstream limits PR creation to collaborators | fixes ready on fork branches |

(Everything here is public on purpose: the tests we ran, the failures we
found, and the ones we got wrong. If a maintainer wants to re-run anything,
`run.sh` in each target directory is the whole recipe.)
