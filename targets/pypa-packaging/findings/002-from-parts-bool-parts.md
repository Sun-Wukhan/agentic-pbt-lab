# Finding 002: `Version.from_parts` / `__replace__` validators accept `bool` and manufacture unparseable versions

- Target / property: pypa/packaging @ 713d0d78 (26.4.dev0) — V8
  (from_parts/__replace__ agreement), found by a manual probe of the
  new 26.1 validators while writing the V8 strategies (the fuzz
  strategies themselves use genuine ints, per the documented contract).
- Minimal counterexample:
  `Version.from_parts(epoch=True, release=(1,))` succeeds and produces
  a version whose `str()` is `'True!1'`; `Version('True!1')` raises
  `InvalidVersion`. Also: `Version.from_parts(release=(True, 2))` has
  `str() == 'True.2'` yet compares and hashes **equal** to
  `Version('1.2')`; `Version('1.2').__replace__(pre=('alpha', True))`
  has `str() == '1.2aTrue'`.
- Observed vs expected:
  - Observed: `bool` parts pass validation everywhere a part is checked
    with `isinstance(x, int)` — `_validate_epoch` (`value or 0` then an
    int check), `_validate_release`, `_validate_pre`, `_validate_post`,
    `_validate_dev` in `src/packaging/version.py`.
  - Expected: the validators reject them with `InvalidVersion`, like
    every other wrong-typed input (`epoch='1'` and `epoch=1.0` both
    raise). `Version.__str__` is documented as "A string representation
    of the version that can be round-tripped", and `from_parts` is
    documented as building a version from validated parts — a product
    that cannot be re-parsed, and whose string form disagrees with its
    equality/hash identity, breaks both contracts.
- Reachability: weaker than Finding 001 and stated plainly: nobody
  writes `epoch=True` on purpose. The exposure is programmatic —
  `from_parts`/`__replace__` (new public API in 26.0/26.1) exist to be
  fed by code that transforms version data (release tooling, index
  software, lock-file generators), where a stray bool from a flag or a
  JSON value is a realistic input class. The failure is silent object
  corruption rather than a loud validation error, which is exactly
  what the validators were added to prevent.
- Verdict: **confirmed bug (low severity)**. The input class is a type
  error, but it is the one type error the validation layer structurally
  cannot see (bool subclasses int), and the result violates the
  class's core round-trip invariant instead of raising the documented
  `InvalidVersion`.
- Upstream issue: not filed (this run is local-only by instruction).
- Fix PR: none. A fix would exclude `bool` explicitly in the five
  `_validate_*` helpers.


**Upstream (2026-11-07):** issue https://github.com/pypa/packaging/issues/1499 · fix PR https://github.com/pypa/packaging/pull/1501
