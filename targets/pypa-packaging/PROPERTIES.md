# Properties — pypa/packaging

Target commit: **713d0d78924ab2686c513dde058ce6aafc63e766** (shallow clone in
`upstream/`, HEAD of `main` on 2026-11-07) · packaging **26.4.dev0** built
from that tree · Python 3.12.3 · Hypothesis 6.170.15 · pytest 9.2.0
Framework: Python, [Hypothesis](https://hypothesis.readthedocs.io/)
Harness layout: plain pytest files under `harness/` run against a pinned
virtualenv (`.venv/` in this directory) into which the cloned package was
installed (`pip install ./upstream`), so the code under test is byte-for-byte
the pinned commit's `src/packaging/`.

Why this target is interesting *now*: the 26.x line rewrote three engines
that used to be simple loops — `Version` ordering is a new hand-rolled flat
key (`version.py::_cmpkey`), specifier matching is a new interval-range
engine with a fast path that must agree with it (`specifiers.py::_fast_match`
vs `packaging._ranges`), and 26.3 added a public set-algebra type
(`ranges.VersionRange`) whose operations must agree with `SpecifierSet`
membership. Every cross-implementation agreement below is a place where the
rewrite can silently diverge from PEP 440 while each half looks right alone.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| V1 | `Version(str(v)) == v` and `str(Version(str(v))) == str(v)` | `Version.__str__`: "A string representation of the version that can be round-tripped" (`version.py`) | `test_v1_str_round_trip` |
| V2 | Messy-but-legal spellings (leading `v`, upper case, `-`/`_`/`.` separators, `alpha`/`beta`/`preview`/`rev`/`r` aliases, implicit 0 numbers) parse to the same `Version` (eq + hash) as the canonical spelling | PEP 440 normalisation rules; `version.py::_LETTER_NORMALIZATION`, `normalize_pre` | `test_v2_messy_spelling_same_version` |
| V3 | Trichotomy and operator coherence: for all a, b exactly one of `a<b`, `a==b`, `a>b` holds; `<=` ⟺ `< or ==`, `>=` ⟺ `> or ==`, `!=` ⟺ `not ==` | `_BaseVersion` docstring contract: instances "can be compared and sorted using the standard Python interfaces"; all six operators key off the same `_key` | `test_v3_trichotomy_and_operator_coherence` |
| V4 | Transitivity: `a<=b and b<=c ⟹ a<=c` (and the `<`/`==` analogues); `sorted()` of a shuffled list is pairwise non-decreasing and a permutation of the input | Same total-order contract; `version.py::_cmpkey` builds a plain tuple key, so the order must be a true total order | `test_v4_transitivity_and_sorting` |
| V5 | `a == b ⟹ hash(a) == hash(b)`; `hash` is stable across calls (key/hash caches must not corrupt it) | `Version.__hash__` / `_key` caching in `version.py` | `test_v5_hash_eq_consistency` |
| V6 | PEP 440 family order around one base release R: `R.devN < RaN < RbN < RrcN < R < R.postN`; `R+x < R+y` for x<y numeric locals; alphanumeric local segments sort before numeric ones; `R < R+local`; any epoch beats every no-epoch version | PEP 440 ordering rules, restated in `version.py::_cmpkey`'s docstring | `test_v6_pep440_family_order` |
| V7 | Attribute coherence: `is_prerelease == (pre is not None or dev is not None)`; `public` is `str(v)` minus the local segment and re-parses to `v.__replace__(local=None)`; `base_version` re-parses to `v` with pre/post/dev/local cleared; `major`/`minor`/`micro` are the release tuple zero-padded to 3 | Property docstrings in `version.py` (`public`, `base_version`, `is_prerelease`, …) | `test_v7_attribute_coherence` |
| V8 | `Version.from_parts(parts) == Version(rendered(parts))` (eq, hash, str); `v.__replace__()` returns `v` itself; replacing one part changes exactly that attribute and the result re-parses to itself | `Version.from_parts` / `Version.__replace__` docstrings (26.0/26.1): "build a version without going through a string"; validators `_validate_*` in `version.py` | `test_v8_from_parts_and_replace` |
| V9 | Pickle round-trip: `pickle.loads(pickle.dumps(v)) == v`, same hash, same str | `Version` class docstring: "Instances are safe to serialize with pickle. They use a stable format" (26.2) | `test_v9_pickle_round_trip` |
| V10 | `canonicalize_version(str(v))` equals `str(v)` with trailing release zeros stripped, is idempotent, re-parses to `v`; with `strip_trailing_zero=False` it equals `str(v)` | `utils.canonicalize_version` docstring + PEP 625 note | `test_v10_canonicalize_version` |
| S1 | Constructed witnesses: `>=v`, `<=v`, `==v`, `~=v` all contain `v` itself (prereleases=True); `==v.*` contains every version in v's prefix family; `!=v` does not contain `v`; `>v` contains a constructed greater release and not `v`; `<v` contains a constructed smaller release and not `v` | PEP 440 operator semantics; `specifiers.py` operator table | `test_s1_constructed_witnesses` |
| S2 | `Specifier(s).contains(w, prereleases=p) == SpecifierSet(s).contains(w, prereleases=p)` for p ∈ {True, False, None} — the single-specifier path (`_fast_match` / per-spec ranges) and the set path (fast per-spec loop / intersected bounds / filter fallback) are two implementations of one semantics | `specifiers.py::Specifier.contains`, `SpecifierSet.contains` (three disjoint code paths) | `test_s2_specifier_vs_set_agreement` |
| S3 | `SpecifierSet.contains(w, prereleases=p) == (w ∈ SpecifierSet.filter([w], prereleases=p))` for p ∈ {True, False}, and for p=None on a lone item | `SpecifierSet.contains` slow path is literally `bool(list(self.filter([check_item])))`; the fast path must not diverge from it | `test_s3_contains_vs_filter_single` |
| S4 | Filter semantics over a list, p=None (PEP 440 rule): filter output ⊆ input (subsequence, order preserved); every *final* in the output is contained; a contained final in the input always appears; contained pre-releases appear iff no contained final is in the input | `filter` docstrings (26.0 versionchanged note) + `_pep440_filter_prereleases` / `_ranges.filter_by_ranges` | `test_s4_filter_list_semantics` |
| S5 | `SpecifierSet(a) & SpecifierSet(b)` contains w (prereleases=True) iff both operands do; `str` of the combination re-parses to an equal set with identical sampled membership | `SpecifierSet.__and__` docstring | `test_s5_and_combination` |
| S6 | String round-trips: `SpecifierSet(str(ss)) == ss` with equal hash; `Specifier(str(sp)) == sp` with equal hash | `__str__` docstrings: "A string representation … that can be round-tripped" | `test_s6_str_round_trip` |
| S7 | Pre-release policy: with `prereleases=False` no pre-release is ever contained; a specifier naming a pre-release (`>=1.0a1`) autodetects and contains its own bound; `filter` with p=None yields a matching lone pre-release but drops it when a matching final is present in the same list | PEP 440 pre-release rules; `Specifier.prereleases` property; 26.0 `contains`/`filter` docstring changes | `test_s7_prerelease_policy` |
| S8 | `is_unsatisfiable` soundness (default-configured sets): unsatisfiable ⟹ no sampled version is contained even at prereleases=True; a contained witness (prereleases=True) ⟹ not unsatisfiable | `SpecifierSet.is_unsatisfiable` docstring + `_ranges.range_is_empty` / `ranges_are_prerelease_only` | `test_s8_unsatisfiable_soundness` |
| S9 | `===` is case-insensitive literal string equality, not version equality: `===1.0` contains `"1.0"` and `Version("1.0")` but not `"1.0.0"`; `===AbC` contains `"aBc"` | PEP 440 arbitrary-equality semantics; `specifiers.py` `===` handling | `test_s9_arbitrary_equality` |
| S10 | Range algebra agrees with membership (sets built with prereleases=True on both sides, no `===`): `w ∈ (A & B) ⟺ w∈A ∧ w∈B`; `w ∈ (A \| B) ⟺ w∈A ∨ w∈B`; `w ∈ ~A ⟺ w∉A`; `w ∈ (A - B) ⟺ w∈A ∧ w∉B`; and `A.to_range().to_specifier_set()` (when not None) has the same sampled membership as A | `ranges.VersionRange` docstring (26.3): membership, set operations and conversion back "follow" the originating set | `test_s10_range_algebra` |
| S11 | Relation soundness by sampling: `A.is_subset(B)` ⟹ no sampled w with `A.contains(w, prereleases=True)` and not `B.contains(w, prereleases=True)`; `A.is_disjoint(B)` ⟹ no sampled w contained in both; `is_subset`/`is_superset` are reflexive and dual | `SpecifierSet.is_subset/is_superset/is_disjoint` docstrings (26.3) | `test_s11_relation_soundness` |
| S12 | Specifier equality is canonical: `Specifier("==1.0") == Specifier("==1.0.0")` (and hash); `SpecifierSet` equality and hash are insensitive to spec order and duplication | `Specifier.__eq__` docstring example (`==1.2.3` vs `== 1.2.3.0`); `SpecifierSet._canonical_specs` dedup/sort | `test_s12_canonical_equality` |
| R1 | `Requirement(str(r)) == r`, equal hash, and `str` is a fixed point | `Requirement.__str__`/`__eq__`/`__hash__` (22.0/23.2/26.3 notes) | `test_r1_requirement_round_trip` |
| R2 | Name/extra spelling is presentation-only: requirements differing only in name/extra case or `-`/`_`/`.` spelling compare and hash equal | `Requirement.__eq__` (26.3: "Equality and hashing normalize requirement names, extras, and equivalent specifiers") + PEP 685 | `test_r2_requirement_spelling_invariance` |
| R3 | Round-trip preserves parts: re-parsed `.specifier`, `.marker`, `.url`, `.extras` equal the original's | `Requirement.__init__` field construction from `_parser` output | `test_r3_requirement_parts_preserved` |
| R4 | Marker evaluation survives the round-trip: `Requirement(str(r)).marker.evaluate(env) == r.marker.evaluate(env)` for generated environments | Composition of R1 with `Marker.evaluate` | `test_r4_requirement_marker_evaluation` |
| M1 | `Marker(str(m)) == m`, equal hash, `str` a fixed point | `Marker.__eq__`/`__hash__` are defined on `str(self)`; `_format_marker` output must re-parse to the same string | `test_m1_marker_round_trip` |
| M2 | `(a & b).evaluate(env) == a.evaluate(env) and b.evaluate(env)`; `(a \| b)` likewise; combined markers re-parse equal (`Marker(str(a & b)) == a & b`) | `Marker.__and__`/`__or__` docstrings (26.1) + `_evaluate_markers` group semantics | `test_m2_marker_and_or_algebra` |
| M3 | Tautology/contradiction: for a string key k and value v, `k == "v" or k != "v"` evaluates True and `k == "v" and k != "v"` evaluates False under every generated environment | `_evaluate_markers`: `or` splits groups, result is `any(all(group))` | `test_m3_marker_tautology` |
| M4 | Version-key marker comparisons agree with `Specifier`: `python_version <op> "v"` evaluated at `python_version=w` equals `Specifier(f"{op}{v}").contains(w, prereleases=True)`, including `==` with `.*` prefixes | `markers.py::_eval_op` builds exactly that `Specifier` for `MARKERS_REQUIRING_VERSION` keys — the property checks the wiring (lhs/rhs order, env lookup) around it | `test_m4_marker_version_comparisons` |
| M5 | Membership markers: `"x" in os_name` evaluates to the Python substring test on the environment value; `"x" not in os_name` its negation | PEP 508 `in` semantics; `_operators["in"]` | `test_m5_marker_membership` |
| M6 | Extra normalisation (PEP 685): `extra == "Foo_Bar"` and `extra == "foo-bar"` are equal markers, and both evaluate True against `extra="FOO.bar"` in the requirement context | `markers.py::_normalize_extras`, `_normalize`, `Marker.evaluate` extra canonicalisation | `test_m6_marker_extra_normalization` |
| M7 | Sanity: markers over the standard keys evaluate against `default_environment()` without raising, and return a `bool` | `Marker.evaluate` docstring contract | `test_m7_marker_default_environment` |
| U1 | `canonicalize_name` matches the PEP 503 reference `re.sub(r"[-_.]+", "-", n).lower()` on every generated name-like string, and is idempotent | `utils.canonicalize_name` comment: "Emulates `re.sub(...)` from PEP 503" — the hand-rolled loop must match the regex it replaces | `test_u1_canonicalize_name_model` |
| U2 | For valid names n: `is_normalized_name(canonicalize_name(n))` is True; `is_normalized_name(s)` implies `canonicalize_name(s) == s` whenever s is a valid name | `utils.is_normalized_name` docstring, incl. its leading/trailing-hyphen caveat (scoped to valid names accordingly) | `test_u2_is_normalized_name` |
| N1 | No-crash: feeding generated text (structured fragments + noise) to `Version`, `Specifier`, `SpecifierSet`, `Requirement`, `Marker` either succeeds or raises exactly the documented exception (`InvalidVersion` / `InvalidSpecifier` / `InvalidRequirement` / `InvalidMarker`) — never any other exception type | Each constructor's `:raises:` contract; the Maaz et al. no-crash property class | `test_n1_parsers_raise_only_documented` |

## Notes

- **Version strategy** builds strings from parts (epoch / release / pre /
  post / dev / local) with every legal spelling variant, plus a *messy*
  rendering of the same parts (aliases, separators, case, leading `v`), so
  V2 compares two renderings of one logical version. Specifier tests draw
  spec versions and candidate versions from a shared pool of releases so
  family collisions (same epoch + trimmed release — the `_fast_match`
  carve-out) are dense rather than accidental.
- **Scoped out by design** (documented behaviour, would only re-derive it):
  `contains` with `prereleases=None` accepting a lone pre-release while
  `filter` over a mixed list drops it (26.0 docstring change — S3/S4 encode
  exactly the documented split instead); marker ordering comparisons
  (`<`, `>`) on non-version keys degrading to False/equality
  (`markers.py::_operators`, long-standing); PEP 440 local-version ordering
  (alphanumeric before numeric) as a *quirk* — it is in V6 as spec, not as a
  bug; `LegacyVersion` (removed in 22.0 — invalid strings raise
  `InvalidVersion`, covered by N1).
- **`from_parts`/`__replace__` strategies use genuine `int`s only.** The
  validators' documented contract takes ints; `bool` (an `int` subclass)
  inputs are probed once, separately, in `findings/` triage rather than
  fuzzed, so a verdict there is a judgement call about an undocumented
  input class, not noise in V8.
- Marker environments are generated as full dictionaries over the standard
  keys plus `extra`/`extras`/`dependency_groups` where the property needs
  them, so `UndefinedEnvironmentName` never fires spuriously.
