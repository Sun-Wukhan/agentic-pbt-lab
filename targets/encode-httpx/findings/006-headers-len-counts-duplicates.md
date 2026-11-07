# Finding: `len(Headers)` counts raw entries including duplicates, while `keys()`/`items()`/iteration count distinct keys

- Target / property: encode/httpx `main` @ `b5addb6` — property H2 (Mapping length consistency). Any duplicate key is a counterexample; the minimal one is two entries.
- Minimal counterexample (plain Python, see `reproducers.py`):
  - `h = Headers([("a", "1"), ("a", "2")])`
  - `len(h)` → `2`
  - `list(h.keys())` → `["a"]`, `len(list(h))` → `1`, `dict(h)` → `{"a": "1, 2"}`
  - So `len(h) != len(list(h))`: code sizing a dict by `len(headers)`, or checking `len(headers) == 0` after deletions by key, or zipping `headers` with `range(len(headers))`, misbehaves.
- Observed vs expected: `Headers` subclasses `typing.MutableMapping`, whose contract makes `__len__` the number of keys the mapping yields (the same count `__iter__`, `keys()` and `items()` expose — all of which deduplicate here, with `items()` concatenating duplicate values per RFC 7230 §3.2.2, exactly as `__getitem__` documents). Only `__len__` (and `copy` bookkeeping via `_list`) uses the raw entry count, via `return len(self._list)`. No upstream test pins the duplicate case (checked `tests/models/test_headers.py` and `tests/client/test_headers.py`: no `len()` assertion over duplicate-key headers).
- Verdict: **confirmed bug** (internal inconsistency with the class's own Mapping contract; low severity).
- Reasoning: one of the two behaviours has to be the odd one out, and it is `__len__`: every other size- or key-oriented view on the class — iteration, `keys()`, `items()`, `values()`, `dict(h)`, `bool(h)` via `__len__`… (`bool` uses `__len__`, so `bool` agrees with the raw count while iteration disagrees) — treats the header set as a mapping from distinct case-folded keys to concatenated values. Multi-value storage is an implementation detail exposed deliberately through `multi_items()`/`get_list()`/`raw`, not through the Mapping interface. Reachability is generic mapping-consumer code: `dict(headers)`, `{**headers}`, logging summaries, and test assertions like `assert len(response.headers) == 3` after a server sends a duplicated header (e.g. two `Set-Cookie`-style or appended `Vary` entries) all observe the inconsistency.
- Upstream issue: none (local-only run; no upstream contact per run instructions).
- Fix PR: n/a.
