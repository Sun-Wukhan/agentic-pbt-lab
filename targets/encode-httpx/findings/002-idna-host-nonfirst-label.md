# Finding: `URL.host` only IDNA-decodes when the *first* label is punycode — `www.中国…` hosts come back still encoded

- Target / property: encode/httpx `main` @ `b5addb6` — property U4 (IDNA contract in every label position). Shrunk counterexample (Hypothesis): labels `["0", "ß"]` → `URL("http://0.ß/").host == '0.xn--zca'`, expected `'0.ß'`.
- Minimal counterexample (plain Python, see `reproducers.py`):
  - `URL("http://中国.icom.museum/").host` → `"中国.icom.museum"` (correct)
  - `URL("http://www.中国.icom.museum/").host` → `"www.xn--fiqs8s.icom.museum"` (still punycode)
  - `URL("http://www.xn--fiqs8s.icom.museum/").host` → `"www.xn--fiqs8s.icom.museum"` (same)
  - `URL("http://www.中国.icom.museum/").raw_host` → `b"www.xn--fiqs8s.icom.museum"` (correct) — so the unicode spelling a user constructs with is not the host they read back.
- Observed vs expected: the `URL.host` docstring states the host is "normalized to lowercase, with IDNA hosts decoded into unicode", and the class docstring: "Internationalized domain names are represented in unicode, without IDNA encoding applied." Upstream's own `test_idna_url` asserts `url.host == <unicode host>` for six IDNA URLs — but every parametrized case places the internationalized label first, so the gap is uncovered upstream.
- Verdict: **confirmed bug**.
- Reasoning: `URL.host` in `httpx/_urls.py` is `if host.startswith("xn--"): host = idna.decode(host)` — the check is on the whole host string rather than per label, so decoding is skipped whenever any non-punycode label (e.g. `www`) precedes the A-label. Reachability: subdomains in front of IDN labels are the common case (`www.`, `api.`), and `.host` is consumed by user allowlist/origin code and internally by `httpx._utils.URLPattern` (proxy/no-proxy matching builds its regex from `url.host`) and `httpx._client` redirect handling (`url.copy_with(host=request.url.host)`), so a pattern written with the unicode host will not match as intended. The fix direction is to decode whenever any label starts with `xn--` (i.e. attempt `idna.decode` for hosts containing an `xn--` label), which is also what makes Finding 003's handling necessary.
- Upstream issue: none (local-only run; no upstream contact per run instructions).
- Fix PR: n/a.
