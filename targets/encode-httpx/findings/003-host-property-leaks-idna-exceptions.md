# Finding: reading `.host` on a successfully parsed URL can raise raw `idna` exceptions (`IDNAError`, `InvalidCodepoint`, `UnicodeError`)

- Target / property: encode/httpx `main` @ `b5addb6` — property U5 (host-access safety). Shrunk counterexamples (Hypothesis): hosts built from labels `xn--a`, `xn--` in first position.
- Minimal counterexample (plain Python, see `reproducers.py`):
  - `URL("http://xn--a.com/")` constructs fine; `.host` raises `idna.core.InvalidCodepoint: Codepoint U+0080 at position 1 of '\x80' not allowed`
  - `URL("http://xn--.com/")` constructs fine; `.host` raises `idna.core.IDNAError: Malformed A-label, no Punycode eligible content found`
  - (`URL("http://www.xn--a.com/")` does not raise — it returns the still-encoded host; that is Finding 002's side of the same property.)
- Observed vs expected: the constructor accepts these hosts (they are syntactically valid ASCII reg-names to `encode_host`, which never IDNA-decodes), so a `URL` object exists and passes `is_absolute_url`, equality, printing — but the documented `.host` accessor then detonates with a third-party exception type that callers have no reason to catch. httpx's own error contract is `httpx.InvalidURL`, raised at parse time for invalid IPv4/IPv6/IDNA inputs on the *encode* path (`encode_host` wraps `idna.IDNAError` into `InvalidURL` for non-ASCII hosts).
- Verdict: **confirmed bug**.
- Reasoning: `URL.host` (`httpx/_urls.py`) calls `idna.decode(host)` with no error handling whenever the stored host starts with `xn--`. Parse-time validation and property-time decoding disagree about what a valid A-label is: anything shaped like `xn--…` in first position is stored un-validated and decoded lazily. Reachability: `_utils.URLPattern.__init__` reads `url.host` unconditionally (proxy configuration built from an unusual URL crashes client construction), `_client.py` reads `request.url.host` when resolving relative redirect targets, and any user logging/allowlist code touching `.host` on an attacker-influenced URL (e.g. a redirect target or webhook URL) gets an unhandled exception instead of a value or an `InvalidURL`. A fix either validates A-labels in `encode_host` (rejecting with `InvalidURL`, symmetric with the non-ASCII path) or catches `idna` errors in the property.
- Upstream issue: none (local-only run; no upstream contact per run instructions).
- Fix PR: n/a.
