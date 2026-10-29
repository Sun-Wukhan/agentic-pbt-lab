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

(Everything here is public on purpose: the tests we ran, the failures we
found, and the ones we got wrong. If a maintainer wants to re-run anything,
`run.sh` in each target directory is the whole recipe.)
