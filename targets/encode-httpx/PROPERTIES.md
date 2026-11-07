# Properties — encode/httpx

Target version: **httpx `main` @ `b5addb64f0161ff6bfe94c124ef76f6a1fba5254`**
(post-0.28.1 development tree, shallow clone in `./httpx`, installed into the
venv — `httpx.__version__` still reports 0.28.1) ·
Python 3.12.3 · Hypothesis 6.170.14 · pytest 9.2.0
Framework: Python, [Hypothesis](https://hypothesis.readthedocs.io/)
Harness layout: plain pytest files under `harness/` run against the clone
installed into the shared lab virtualenv (`~/workspace/.venv-pbt`, override
with `PBT_VENV`). All properties are hermetic: URLs, `QueryParams` and
`Headers` only — no transports, no network.

Surfaces were chosen by reading `httpx/_urlparse.py`, `httpx/_urls.py`,
`httpx/_models.py` (`Headers`) and `httpx/_utils.py` first, per the lab
method. `_urlparse.py`'s module docstring is the load-bearing contract:
it claims validation and normalization "as described by RFC3986",
differentiation of empty vs absent query strings, and scheme/hostname/port/
path normalization with IDNA support.

## URL properties (`harness/test_pbt_urls.py`)

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| U1 | Parse/print fixpoint: `str(URL(str(u))) == str(u)` and `URL(str(u)) == u` for component-built URLs | `_urlparse.urlparse`: "The parsed ASCII bytestrings are our canonical form. All properties of the URL are derived from these." | `test_u1_parse_print_fixpoint` — **FAILS: Finding 001** |
| U2 | Normalization idempotence on mutated URL strings: for any string the constructor accepts, `str(URL(s))` re-parses to itself; rejected strings raise only `InvalidURL` | Same canonical-form claim, applied to the mutation strategy | `test_u2_mutation_idempotence_and_no_crash` — **FAILS: Finding 005** |
| U3 | Scheme and host are lowercased; a default port (80/http, 443/https, 80/ws, 443/wss, 21/ftp) normalizes to `None` and compares equal to the port-less URL | `URL` class docstring ("normalized to always be lowercased", port normalization) + `_urlparse.normalize_port` | `test_u3_case_and_default_port_normalization` — **FAILS: Finding 001** |
| U4 | IDNA contract in **every** label position: for a host built from unicode labels, `url.host` is the unicode host, `url.raw_host` is its IDNA encoding, and the unicode and punycode spellings construct equal URLs | `URL.host` docstring ("Internationalized domain names are represented in unicode, without IDNA encoding applied") + upstream `test_idna_url`, whose parametrized cases only ever place the IDNA label first | `test_u4_idna_host_any_label_position` — **FAILS: Finding 002** |
| U5 | Host-access safety: for any ASCII host the constructor accepts, reading `.host` either returns a `str` or raises `httpx.InvalidURL` — never a raw `idna` exception | Constructor accepts the host (parse-time validation in `encode_host`); nothing documents property access raising third-party exceptions | `test_u5_host_access_never_leaks_idna_errors` — **FAILS: Finding 003** |
| U6 | Dot-segment removal matches the RFC 3986 §5.2.4 `remove_dot_segments` algorithm (model implementation in the harness) for absolute paths | `_urlparse` module docstring ("normalization as described by RFC3986"); `normalize_path` docstring cites §5.2.4 | `test_u6_dot_segments_match_rfc3986` — **FAILS: Finding 004** |
| U7 | `copy_with` fidelity: changing one component leaves every other raw component untouched; `copy_with()` is the identity | `URL.copy_with` docstring ("returning a new URL with some components altered") | `test_u7_copy_with_component_fidelity` — **FAILS: Finding 001** (via the port) |
| U8 | `join` resolution semantics: joining an absolute reference returns it (normalized); joining `""` returns the base without its fragment; joining `#frag` changes only the fragment; joining `?q` changes only query and fragment. Plus the fixed RFC 3986 §5.4 example table (normal + abnormal) against base `http://a/b/c/d;p?q` | `URL.join` docstring ("Return an absolute URL, using this URL as the base") + RFC 3986 §5.4, which §5.2 resolution is defined by | `test_u8_join_metamorphic`, `test_u8_join_rfc3986_examples` — passes with two documented carve-outs for present-but-empty query/fragment components, where `URL.join` inherits `urllib.parse.urljoin` behaviour (Finding 007, wrong property as originally stated) |
| U9 | Percent-encoding round-trips: a path/query/fragment component string without `%` comes back decoded-equal via `.path` / `.params` / `.fragment`, and `raw_path` re-parses to the same URL | `_urlparse.quote` (percent-encoding omitting existing escapes) + `URL.path`/`raw_path` docstrings | `test_u9_percent_encoding_round_trip` |
| U10 | Empty vs absent query is distinguished: `URL("http://e/?") != URL("http://e/")` as strings, `copy_with(query=None)` drops the `?`, `raw_path` reflects the distinction | `_urlparse` module docstring: "It properly differentiates between an empty querystring and an absent querystring, to distinguish URLs with a trailing '?'" | `test_u10_empty_vs_absent_query` |
| U11 | Equality/hash consistency: `u == str(u)`; equal-after-normalization URLs hash equally; `URL(u) == u` | `URL.__eq__`/`__hash__` are both defined over the normalized string form | `test_u11_equality_and_hash_consistency` — **FAILS: Finding 001** |

## QueryParams properties (`harness/test_pbt_queryparams.py`)

| # | Property (invariant) | Source | Test |
|---|----------------------|--------|------|
| Q1 | String round-trip: `QueryParams(str(q)) == q` | `QueryParams.__str__` (urlencode of `multi_items()`) paired with the `parse_qs` constructor path | `test_q1_str_round_trip` |
| Q2 | Multi-value preservation: the multiset of `multi_items()` survives the string round-trip, and `get_list(k)` preserves insertion order | `multi_items`/`get_list` docstrings ("Allow duplicate keys to occur") | `test_q2_multi_value_preservation` |
| Q3 | `set`: `q.set(k, v).get_list(k) == [coerce(v)]`, other keys untouched, original unchanged | `QueryParams.set` docstring + immutability note in `update()` ("QueryParams are immutable since 0.18.0") | `test_q3_set_semantics` |
| Q4 | `add`: `q.add(k, v).get_list(k) == q.get_list(k) + [coerce(v)]`, original unchanged | `QueryParams.add` docstring | `test_q4_add_semantics` |
| Q5 | `remove`: key absent afterwards; removing a missing key is the identity; original unchanged | `QueryParams.remove` docstring | `test_q5_remove_semantics` |
| Q6 | `merge` replaces wholesale: merged keys take the other instance's full value list, untouched keys survive, merge is associative under `==`, empty merge is identity | `QueryParams.merge` docstring (worked examples show replacement, e.g. `a=123` merged with `a=456` gives `a=456`) | `test_q6_merge_semantics` |
| Q7 | Primitive coercion: `True→"true"`, `False→"false"`, `None→""`, other values via `str()` — for dict, pair-list, `set` and `add` construction paths | `_utils.primitive_value_to_str` + the coercion comment in `QueryParams.__init__` | `test_q7_primitive_coercion` |
| Q8 | URL integration: `URL("http://e/", params=q).params == q`, and `copy_set_param`/`copy_add_param`/`copy_remove_param`/`copy_merge_params` agree with the corresponding `QueryParams` operations | `URL.__init__` params handling + `copy_*_param` wrappers in `_urls.py` | `test_q8_url_params_integration` |

## Headers properties (`harness/test_pbt_headers.py`)

| # | Property (invariant) | Source | Test |
|---|----------------------|--------|------|
| H1 | Case-insensitive lookup: any case permutation of a key gives the same `__getitem__`, `get_list`, `in` result, and `del` removes every case variant | `Headers` class docstring: "HTTP headers, as a case-insensitive multi-dict" | `test_h1_case_insensitive_lookup` |
| H2 | Mapping length consistency: `len(h) == len(list(h.keys())) ==` number of distinct case-folded keys | `Headers` subclasses `typing.MutableMapping`, whose contract ties `__len__` to the keys yielded by `__iter__`; `keys()`/`items()`/`__iter__` here all deduplicate | `test_h2_len_matches_distinct_keys` — **FAILS: Finding 006** |
| H3 | Concatenated reads agree: `h[k] == ", ".join(h.get_list(k))`, and `items()`/`values()` expose the same concatenation | `__getitem__` docstring (concatenate per RFC 7230 §3.2.2) + `items()` docstring | `test_h3_concatenated_reads_agree` |
| H4 | Reconstruction fidelity: `Headers(h.multi_items()) == h`, `Headers(h.raw) == h`, and `raw` preserves the exact key casing given at construction | `multi_items` docstring; `raw` property ("the raw header items, as byte pairs") | `test_h4_reconstruction_fidelity` |
| H5 | `__setitem__` semantics: afterwards `get_list(k) == [v]`, duplicates removed, a replaced key keeps its first-occurrence position, a new key is appended | `__setitem__` docstring ("removing any duplicate entries. Retains insertion order") | `test_h5_setitem_semantics` |
| H6 | `update` semantics: keys present in the argument take the argument's full value list, other keys are untouched, `update` with an equal instance is the identity | `Headers.update` implementation contract (pop-then-extend), mirroring `MutableMapping.update` | `test_h6_update_semantics` |
| H7 | `get_list(split_commas=True)` equals flattening `get_list()` on commas with whitespace stripped | `get_list` docstring | `test_h7_get_list_split_commas` |
| H8 | Equality: insensitive to entry order and key case, sensitive to values and to duplicate counts | `Headers.__eq__` (sorted raw pairs with lowercased keys) | `test_h8_equality_semantics` |

## Notes

- Strategies build URLs from components (schemes from a small set including
  mixed case; host labels including unicode IDNA labels, pre-encoded `xn--`
  labels, IPv4/IPv6 literals; ports including scheme defaults; path segments
  including `.`/`..`/empty/percent-escapes/unicode) and additionally mutate
  valid URL strings character-wise. ASCII control characters are excluded
  from generation: `_urlparse` documents them as `InvalidURL`, so they
  would only re-derive the documented rejection.
- Ports are generated in `0..65535` for the main strategies. Out-of-range
  ports are accepted by the parser (no range check in `normalize_port`);
  that leniency matches stdlib `urlsplit` parse-time behaviour and is
  recorded as an observation in `findings/`, not counted as a property
  failure.
- QueryParams keys/values are generated from an alphabet with the query
  delimiters (`&`, `=`, `+`, `%`, `;`, `#`-free) plus unicode, because
  `parse_qs`/`urlencode` asymmetries — if any — live exactly there.
- Header keys are generated as RFC token-like strings with randomized case;
  values include commas (for the concatenation/split properties) and
  leading/trailing spaces are avoided, since HTTP field-value optional
  whitespace is a separate normalization contract.
