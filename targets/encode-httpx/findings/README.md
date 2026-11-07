# Findings index — encode/httpx (main @ b5addb6)

Runs: 2026-11-07, Hypothesis 6.170.14, 500 examples per property
(default profile), then 2,000 examples per property (long profile) —
identical outcomes at both sizes; see `../results/`.
27 properties (65 test items including the 37-case RFC 3986 §5.4 table):
18 properties pass clean; 9 property failures, triaged into
6 confirmed bugs and 1 wrong-property observation below.

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | Default-port elision depends on scheme case: `URL("HTTP://example.org:80/") != URL("http://example.org/")`, hashes differ, first print is not a fixpoint, `copy_with` silently drops the port | U1, U3, U7, U11 | **Confirmed bug** — `urlparse()` passes the un-lowercased scheme to `normalize_port` |
| 002 | `URL.host` IDNA-decodes only when the *first* label is punycode: `www.中国…` hosts read back still encoded | U4 | **Confirmed bug** — `startswith("xn--")` on the whole host instead of per-label decoding; contradicts the documented `.host` contract |
| 003 | Reading `.host` on a parsed URL raises raw `idna` exceptions (`IDNAError`, `InvalidCodepoint`, `UnicodeError`) for malformed A-labels like `xn--a.com` | U5 | **Confirmed bug** — lazy decode in the property with no `InvalidURL` wrapping; parse-time validation and decode-time validation disagree |
| 004 | Dot-segment removal loses the trailing slash RFC 3986 §5.2.4 preserves: `"/.."`, `"/a/.."` → `""`, `"/a/."` → `"/a"` | U6 | **Confirmed bug** (spec divergence, modest severity) — `normalize_path` deletes a trailing dot segment instead of replacing it with `/` |
| 005 | A leading `:` is eaten as an empty-scheme delimiter: `URL(":http://e/")` is a relative URL whose string form re-parses as an absolute URL with host `e`; `validate_path`'s own leading-`:` guard is bypassed | U2 | **Confirmed bug** (parser edge) — `URL_REGEX`'s scheme group matches empty before the colon |
| 006 | `len(Headers)` counts raw entries incl. duplicates; `keys()`/`items()`/iteration count distinct keys | H2 | **Confirmed bug** (internal inconsistency with the class's own `MutableMapping` contract; no upstream test pins the duplicate case) |
| 007 | `URL.join` flattens present-but-empty query/fragment, keeps the base fragment on an empty reference, and returns references unresolved for schemes outside urllib's tables | U8 | Wrong property as stated — all three behaviours are inherited from `urllib.parse.urljoin`, which `join` delegates to by construction; recorded as an observation, test now encodes the delegation contract |

Passed clean (2,000 examples each): U8 (with Finding 007's carve-outs),
U9 (percent-encoding round-trips), U10 (empty vs absent query — including
`copy_with(query=None)`), the full RFC 3986 §5.4 join table (37 cases),
Q1–Q8 (QueryParams string round-trips, multi-value preservation and order,
set/add/remove/merge semantics incl. merge associativity, primitive
coercion on all construction paths, URL `params` integration),
H1, H3–H8 (Headers case-insensitive lookup in all case permutations,
concatenated reads, reconstruction via `multi_items()`/`raw` with casing
preserved, `__setitem__` position/duplicate semantics, `update`
replacement semantics, `get_list(split_commas=True)`, and order/case
-insensitive but duplicate-sensitive equality).

Observation (not a property failure): out-of-range ports are accepted at
parse time (`URL("http://e.org:99999/").port == 99999`, likewise `":+80"`
is accepted and elided). `normalize_port` performs no range check. This
matches stdlib `urlsplit` parse-time leniency, so it is recorded here
rather than counted as a bug; the failure would surface at connect time.

Reproduction code for all findings: `reproducers.py` (plain Python, no
Hypothesis; output in `../results/reproducers.log`). Full logs:
`../results/run-500.log`, `../results/run-2000.log`
(`run-500-preliminary.log` is the first 500-example pass, kept because it
is the run whose U8 counterexamples produced Finding 007's carve-outs).
Nothing was filed upstream — this run was local-only by instruction.

## Upstream (2026-11-07)

encode/httpx limits PR creation to collaborators, so each confirmed bug was
raised as a "Potential Issue" discussion with its tested fix branch linked:

- 001 (scheme case / port): https://github.com/encode/httpx/discussions/3817 — branch `fix/url-scheme-case-port`
- 002+003 (IDNA `.host`): https://github.com/encode/httpx/discussions/3818 — branch `fix/url-host-idna-labels`
- 004 (dot segments): https://github.com/encode/httpx/discussions/3819 — branch `fix/url-normalize-path-rfc`
- 005 (leading colon): https://github.com/encode/httpx/discussions/3820 — branch `fix/url-leading-colon`
- 006 (`Headers.__len__`): https://github.com/encode/httpx/discussions/3821 (no fix branch; one-line fix offered)
- 007 (`URL.join`): not a bug — delegates to `urllib.parse.urljoin` by construction; not filed.
