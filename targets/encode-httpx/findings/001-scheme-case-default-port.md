# Finding: default-port elision depends on the case of the scheme — `URL("HTTP://example.org:80/") != URL("http://example.org/")`

- Target / property: encode/httpx `main` @ `b5addb6` — properties U3 (case + default-port normalization), U1 (parse/print fixpoint), U7 (`copy_with` fidelity), U11 (equality/hash consistency). Shrunk counterexample (Hypothesis): components `{scheme: 'HTTP', host: '0', port: 80}` → `URL('http://0:80/?') == 'http://0:80/?'` is `False`.
- Minimal counterexample (plain Python, see `reproducers.py`):
  - `URL("HTTP://example.org:80/")` → `str` is `"http://example.org:80/"`, `.port == 80`
  - `URL("http://example.org:80/")` → `str` is `"http://example.org/"`, `.port is None`
  - `URL("HTTP://example.org:80/") == URL("http://example.org/")` → `False`; hashes differ
  - `str(URL(str(URL("HTTP://example.org:80/"))))` → `"http://example.org/"` — the first print is not a fixpoint
  - `URL("HTTP://example.org:80/").copy_with(path="/x")` → `"http://example.org/x"` — the copy silently drops the port the original carries
  - Same for `Https://…:443/`, `WS://…:80/`, `WSS://…:443/`, `FTP://…:21/`.
- Observed vs expected: the `URL` docstring promises the scheme is "normalized to always be lowercased" and that default ports are "normalized to `None`", with `URL("http://example.com") == URL("http://example.com:80")` as the worked example. With an uppercase scheme the port survives normalization, so two spellings of the same URL compare unequal, hash differently, print differently on first vs second parse, and diverge under `copy_with`.
- Verdict: **confirmed bug**.
- Reasoning: in `httpx/_urlparse.py`, `urlparse()` computes `parsed_scheme = scheme.lower()` but then calls `normalize_port(port, scheme)` with the **raw, un-lowercased** `scheme`; `normalize_port` looks the scheme up in `{"ftp": 21, "http": 80, "https": 443, "ws": 80, "wss": 443}` and misses for `"HTTP"`. The one-line fix is to pass `parsed_scheme`. Reachability is ordinary user code: URLs are routinely constructed from mixed-case input (hand-written config, copied links, `Location` headers preserved verbatim), and URL equality/hash feed redirect handling, request matching in test transports (`httpx.MockTransport` handlers compare URLs), cache keys and allowlist checks — all of which silently treat the two spellings as different origins. No documentation sanctions case-dependent normalization; the docstring asserts the opposite.
- Upstream issue: none (local-only run; no upstream contact per run instructions).
- Fix PR: n/a.
