# Finding: "untrusted key" in T4 was secretly the trusted key

- Target / property: tailscale/tailscale @ 445df50dcf76b6172dc4688c2097c47878f41641, property T4 (`TestPBTTKASignatureNegatives`), wave 2 bring-up at checks=100 (failed after 87 tests).
- Minimal counterexample: rapid draws `salt = 0`, `salt2 = 0`, `sigpos = 0`, `sigxor = 1`. The sub-case "update signed by an untrusted key must be rejected" was accepted by `Inform`.
- Observed vs expected: observed acceptance; the property expected rejection because the signer was supposed to be a key the authority does not trust.
- Verdict: harness artifact (generator key collision, not an upstream bug)
- Reasoning: the harness derives test keys as `ed25519.NewKeyFromSeed(sha256(salt || index))`. `buildOneUpdate` derives its genesis keys with indices 0, 1, 2 under `salt`; T4 derived its "untrusted" key under `salt2` but left the index bytes zeroed — i.e. index 0. When rapid shrank both salts to 0, the "untrusted" key was bit-for-bit the genesis signer key A, so `Inform` correctly accepted its signature. Fix was harness-side: derive the untrusted key with index 77, which can never collide with indices 0–2 even when the salts coincide. After the fix, T4 passes at checks=100 and checks=2000.
- Upstream issue: none (nothing filed, per lab rules)
- Fix PR: none
