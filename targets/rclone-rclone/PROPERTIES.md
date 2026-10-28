# Properties — rclone/rclone

Target commit: `1053463755dea408a4abd42bf2ce917a3884a62a` (master, cloned 2026-10-28)
Framework: Go, [pgregory.net/rapid](https://pgregory.net/rapid) v1.2.0
Harness layout: files under `harness/` mirror their in-repo paths; `run.sh`
copies them into a pristine clone of rclone at the pinned commit, adds
`pgregory.net/rapid` as a test dependency *in that clone only*, and runs them.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| F1 | `SizeSuffix.Set` never panics on arbitrary strings (error is fine) | `fs/sizesuffix.go` — `Set` parses untrusted flag/config input | `TestPBTSizeSuffixSetNeverPanics` |
| F2 | `CountSuffix.Set` never panics on arbitrary strings | `fs/countsuffix.go` — same role as F1 | `TestPBTCountSuffixSetNeverPanics` |
| F3 | Any input `SizeSuffix.Set` *accepts* yields a value ≥ 0, or exactly −1 (the `"off"` sentinel); `Set` itself rejects negatives with "size can't be negative" | `fs/sizesuffix.go` `Set`: the `value < 0` check plus the `"off"` → −1 special case | `TestPBTSizeSuffixSetAcceptedValuesSane` |
| F4 | Same as F3 for `CountSuffix.Set` | `fs/countsuffix.go` `Set`: identical negative check and `"off"` handling | `TestPBTCountSuffixSetAcceptedValuesSane` |
| F5 | `Set(x.stringRoundTrip()) == x` for every `x ≥ 0` | `fs/sizesuffix.go`: "stringRoundTrip turns SizeSuffix into a string which Set parses back to the same value" (relied on by `fs/registry.go` flag defaults and `fs/bwtimetable.go` serialisation) | `TestPBTSizeSuffixRoundTripString` |
| F6 | `Set(x.String()) == x` for every `x ≥ 0` (naive identity, recorded to map where display form and parser diverge) | `fs/sizesuffix.go` `String()` + `Set()` | `TestPBTSizeSuffixStringSetIdentity` |
| F7 | `Set(x.String()) == x` for every `x ≥ 0` (naive identity; `CountSuffix` has no `stringRoundTrip` analogue) | `fs/countsuffix.go` `String()` + `Set()` | `TestPBTCountSuffixStringSetIdentity` |
| S1 | `fspath.Parse` never panics on arbitrary strings | `fs/fspath/path.go` — parses untrusted connection strings | `TestPBTParseNeverPanics` |
| S2 | On success, `Parse` splits losslessly: local ⇒ `Path` is the (slash-converted) input and `ConfigString == ""`; remote ⇒ `ConfigString + ":" + Path` reconstructs the input; re-parsing the reconstruction gives an identical `Parsed` | Doc comment on `Parsed` in `fs/fspath/path.go`; same invariants asserted by the repo's own `fs/fspath/fuzz.go` | `TestPBTParseRoundTrip` |
| L1 | Glob→regexp conversion (`GlobPathToRegexp`, `GlobStringToRegexp`) never panics on arbitrary patterns | `fs/filter/glob.go` — patterns come from user flags/filter files | `TestPBTGlobNeverPanics` |
| L2 | A glob with no metacharacters matches itself | `docs/content/filtering.md` (literal names in filters); `fs/filter/glob.go` escaping of regexp metacharacters | `TestPBTGlobLiteralSelfMatch` |
| L3 | `Filter.Add` / `Filter.AddRule` never panic on arbitrary patterns/rules | `fs/filter/filter.go` `Add`/`AddRule` consume user-supplied rules | `TestPBTFilterAddNeverPanics` |
| C1 | `Cipher.DecryptFileName` / `DecryptDirName` never panic on arbitrary strings, for every name-encryption mode and name encoding | `backend/crypt/cipher.go` — decrypts names read back from real remotes, which may be corrupt or foreign | `TestPBTCipherDecryptNeverPanics` |
| C2 | `DecryptFileName(EncryptFileName(n)) == n` and `DecryptDirName(EncryptDirName(n)) == n` for valid generated names, for every mode/encoding, with directory-name encryption on and off | `backend/crypt/cipher.go` `EncryptFileName`/`DecryptFileName` contract relied on by crypt listings; same round trips as `cipher_test.go`'s fixed cases | `TestPBTCipherEncryptDecryptRoundTrip` |
| C3 | Obfuscation round-trips names containing non-ASCII runes (Latin-1, Latin Extended, Greek, emoji) | `backend/crypt/cipher.go` `obfuscateSegment`/`deobfuscateSegment` range-rotation + quoting rules | `TestPBTObfuscateUnicodeRoundTrip` |
| H1 | `unmarshalSimpleJSON` never panics on arbitrary bytes — including structured JSON with the real field names but extreme values — and never returns `(nil info, nil error)` | `backend/chunker/chunker.go` `unmarshalSimpleJSON` validates metadata found on real remotes ("Be strict about JSON format", strict field checks) | `TestPBTChunkerMetadataNeverPanics` |
| H2 | `unmarshalSimpleJSON(marshalSimpleJSON(size, nChunks, …))` returns the same size and chunk count, flagged as chunker-made | `backend/chunker/chunker.go` `marshalSimpleJSON` ("never produce something you can't consume") | `TestPBTChunkerMetadataRoundTrip` |

## Notes

- Generated "valid names" for C2/C3 deliberately avoid `-`, so no segment can
  resemble an rclone `--versions` suffix (e.g. `-v2024-01-01-...`), which
  `EncryptFileName` intentionally leaves in plain text. That exemption is
  upstream's documented design, not a bug.
- Negative `SizeSuffix`/`CountSuffix` values print as `"off"` and parse back
  to −1, so exact round-trip properties F5–F7 are stated for values ≥ 0 only.
- `unmarshalSimpleJSON` is exercised with a nil `metaObject`: it only stores
  the object in the returned `ObjectInfo` (via `wrapInfo`), and the tests
  never call methods that would dereference it.
