# Finding: dot-segment removal drops the trailing slash RFC 3986 §5.2.4 preserves — `"/.."` and `"/a/.."` normalize to the empty path

- Target / property: encode/httpx `main` @ `b5addb6` — property U6 (dot-segment removal vs the RFC 3986 §5.2.4 model). Shrunk counterexample (Hypothesis): segments `[".."]` → path `"/.."`: httpx stores `""`, the RFC algorithm yields `"/"`.
- Minimal counterexample (plain Python, see `reproducers.py`) — `httpx._urlparse.normalize_path` vs RFC 3986 §5.2.4 `remove_dot_segments`:
  - `"/.."` → httpx `""`, RFC `"/"`
  - `"/."` → httpx `""`, RFC `"/"`
  - `"/a/.."` → httpx `""`, RFC `"/"`
  - `"/a/."` → httpx `"/a"`, RFC `"/a/"`
  - `"/a/b/../.."` → httpx `""`, RFC `"/"`
  - Mid-path cases agree: `"/a/../b"` → `"/b"`, `"/a/./b"` → `"/a/b"`, and trailing-slash forms `"/../"`, `"/a/b/../../"` → `"/"` in both.
  - Consequence at the `URL` level: `URL("http://e.org/a/..") == URL("http://e.org")` and `!= URL("http://e.org/")`, where RFC resolution and browsers produce `http://e.org/`.
- Observed vs expected: `_urlparse.py`'s module docstring claims normalization "as described by RFC3986", and `normalize_path`'s docstring cites §5.2.4 for its algorithm. The RFC algorithm replaces a trailing `/.` or `/..` with `/` (steps 2B/2C treat the complete-input cases by substituting `"/"`), preserving the trailing slash; httpx's segment-list implementation pops the segment and joins, losing it — and can consume the path entirely.
- Verdict: **confirmed bug** (spec divergence; modest severity).
- Reasoning: the implementation splits on `/` and drops `.`/`..` segments, which is equivalent to the RFC algorithm except when a dot segment is the final component: the RFC keeps a trailing `/` because the dot segment is replaced by an (empty) final segment, httpx deletes it outright. The wire impact is masked (`URL.raw_path` substitutes `"/"` for an empty path), but the model impact is real: `.path`, string form, equality and hashing all distinguish `http://e.org` from `http://e.org/`, so a redirect or base-join target ending in `/..` or `/.` produces a URL that compares unequal to the same resource spelled canonically — the exact class of mismatch normalization exists to prevent.
- Upstream issue: none (local-only run; no upstream contact per run instructions).
- Fix PR: n/a.
