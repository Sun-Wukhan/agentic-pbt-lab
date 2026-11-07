# Finding 001: `Requirement` rejects spec-valid strings when a `===` specifier is followed by another specifier containing a space

- Target / property: pypa/packaging @ 713d0d78 (26.4.dev0) — R3/R4
  (requirement round-trip / parts preservation), via the
  `requirement_strings` strategy, which generates PEP 508-valid text.
- Minimal counterexample: `Requirement("x===0,>= 0")` raises
  `InvalidRequirement: Expected comma (within version specifier),
  semicolon (after version specifier) or end`. Equally minimal variants:
  `x===0, >=0`, `x===0,!= 1.0`, `x===0,< 2`, `x (===0,>= 0)`,
  `x===0,>= 0; python_version>'3'`.
- Observed vs expected:
  - Observed: the strings above are rejected. Removing the single space
    (`x===0,>=0`) parses — and `SpecifierSet("===0,>= 0")` accepts the
    spaced form directly. Every other operator tolerates the space
    (`x==0,>= 0` parses), and a `===` in last position is fine
    (`x>=0,===0` parses).
  - Expected: all of them parse to the specifier set `{===0, >=0}`.
    PEP 508's grammar is explicit —
    `version = wsp* ( letterOrDigit | '-' | '_' | '.' | '*' | '+' | '!' )+`
    (the class contains **no comma and no operator characters**),
    `version_one = version_cmp version wsp*`,
    `version_many = version_one (wsp* ',' version_one)*`, and
    "Non line-breaking whitespace is mostly optional with no semantic
    meaning." The library's own grammar docstring in
    `_parser.py::_parse_version_many` states the same rule.
- Mechanism (exact upstream location): `src/packaging/_tokenizer.py`
  `DEFAULT_RULES["SPECIFIER"]` reuses `Specifier._specifier_regex_str`
  from `src/packaging/specifiers.py`, whose `===` alternative matches
  the arbitrary version with `[^\s;)]+` — a run that does **not** stop
  at commas. Inside `_parse_version_many` the token for `===0,>= 0`
  therefore becomes `===0,>=`, the following ` 0` derails the parse, and
  the requirement is rejected. `x===0,>=0` only *appears* to work: the
  token swallows `===0,>=0` whole, and `SpecifierSet`'s comma-split in
  `Requirement.__init__` accidentally reconstructs the intended set.
  The reuse conflates two grammars: for a standalone `Specifier` string
  the greedy class is harmless (a `Specifier` never contains a comma),
  but as a token inside `version_many` it eats the separators the
  parser exists to honour.
- Reachability: `Requirement` is the parser pip and every PEP 508
  consumer (resolvers, metadata readers, lock-file tools) run over
  `Requires-Dist`, `install_requires` and requirements files. `===` is
  the escape hatch used to pin non-PEP-440 version strings (legacy and
  internal indexes); combining it with a second clause and writing the
  conventional space after that clause's operator — the style of
  PEP 508's own headline example (`requests [security,tests] >= 2.8.1,
  == 2.8.*`) — makes the entire requirement unparseable, so the
  distribution's metadata/requirements fail wholesale, with an error
  message pointing at the wrong position.
- Verdict: **confirmed bug**. Acceptance of a requirement flips on an
  insignificant space, contradicting PEP 508 and the parser's own
  documented grammar; the no-space form's success is demonstrably
  accidental (different code path reconstructs it). Not a 26.x
  regression: packaging 25.0 (checked in a scratch venv during triage)
  rejects `x===0,>= 0` the same way — the flaw is inherent to reusing
  the standalone-specifier regex as the requirement tokenizer rule.
- Upstream issue: not filed (this run is local-only by instruction).
- Fix PR: none. A fix would make the tokenizer's `===` version stop at
  a comma when tokenising for the requirement parser (the standalone
  `Specifier` regex can keep its current class).


**Upstream (2026-11-07):** issue https://github.com/pypa/packaging/issues/1500 · fix PR https://github.com/pypa/packaging/pull/1502
