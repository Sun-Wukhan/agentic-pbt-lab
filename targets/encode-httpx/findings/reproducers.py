"""Plain-Python reproducers for the encode-httpx PBT findings.

No Hypothesis required:  python reproducers.py
Each block prints the observed behaviour next to the expected one.
Target: httpx main @ b5addb64f0161ff6bfe94c124ef76f6a1fba5254.
"""
import httpx
from httpx import URL, Headers
from httpx._urlparse import normalize_path

print("== Finding 001: default-port elision depends on scheme case ==")
u = URL("HTTP://example.org:80/")
print("URL('HTTP://example.org:80/') ->", str(u), "port:", u.port)
print("URL('http://example.org:80/') ->", str(URL("http://example.org:80/")), "port:", URL("http://example.org:80/").port)
print("equal to port-less lowercase URL:", u == URL("http://example.org/"), "(expected True)")
print("reparse of str(u):", str(URL(str(u))), "(str(u) itself is not a fixpoint)")
print("copy_with(path='/x'):", str(u.copy_with(path="/x")), "(port silently dropped)")

print()
print("== Finding 002: IDNA decoding only when the first label is punycode ==")
print(URL("http://中国.icom.museum/").host, "(expected 中国.icom.museum)")
print(URL("http://www.中国.icom.museum/").host, "(expected www.中国.icom.museum)")

print()
print("== Finding 003: .host leaks raw idna exceptions ==")
for s in ("http://xn--a.com/", "http://xn--.com/"):
    u = URL(s)
    try:
        print(s, "-> host:", u.host)
    except Exception as e:  # noqa: BLE001 - showing the leaked type is the point
        print(s, "-> .host raises", type(e).__module__ + "." + type(e).__name__, "(expected a value or httpx.InvalidURL)")

print()
print("== Finding 004: dot-segment removal loses the trailing slash ==")
for p, rfc in (("/..", "/"), ("/.", "/"), ("/a/..", "/"), ("/a/.", "/a/"), ("/a/b/../..", "/")):
    print(f"normalize_path({p!r}) -> {normalize_path(p)!r}  (RFC 3986 5.2.4 gives {rfc!r})")
print("URL('http://e.org/a/..') == URL('http://e.org/'):", URL("http://e.org/a/..") == URL("http://e.org/"), "(expected True)")

print()
print("== Finding 005: a leading ':' is silently eaten ==")
u = URL(":http://e/")
print("URL(':http://e/') -> str:", str(u), "| is_absolute_url:", u.is_absolute_url, "| path:", u.path)
print("URL(str(u)) -> host:", URL(str(u)).host, "| is_absolute_url:", URL(str(u)).is_absolute_url, "(same string, different URL)")
print("URL(':foo') ->", str(URL(":foo")), "(colon dropped; expected InvalidURL per validate_path's own rule)")

print()
print("== Finding 006: len(Headers) counts duplicates ==")
h = Headers([("a", "1"), ("a", "2")])
print("len(h):", len(h), "| len(list(h)):", len(list(h)), "| keys:", list(h.keys()), "(Mapping contract: all three should agree)")

print()
print("== Finding 007 (observation): join() flattens empty query/fragment ==")
print("URL('http://e.org/?').join('#frag') ->", str(URL("http://e.org/?").join("#frag")), "(RFC: http://e.org/?#frag)")
print("URL('http://e.org/#a').join('') ->", str(URL("http://e.org/#a").join("")), "(RFC: http://e.org/)")
print("URL('a+b-c.d://0/?a=').join('#frag') ->", str(URL("a+b-c.d://0/?a=").join("#frag")), "(urljoin returns the reference unresolved for unknown schemes)")
