// Property-based tests for the backend/crypt name Cipher.
//
// Lab: Agentic PBT Lab (targets/rclone-rclone) — pristine copy.
// run.sh copies this file into the rclone checkout at
// backend/crypt/pbt_cipher_test.go (in-package, like cipher_test.go,
// because newCipher is unexported). Ciphers use a fixed empty password
// and salt, exactly as cipher_test.go does, so runs are deterministic
// in behaviour (the standard cipher is deterministic; obfuscation uses
// the key, not randomness, for its rotation).
package crypt

import (
	"testing"

	"pgregory.net/rapid"
)

// pbtCiphers builds one fixed test cipher per filename-encryption mode
// (and per name encoding for the standard mode).
func pbtCiphers(t *rapid.T) map[string]*Cipher {
	out := map[string]*Cipher{}
	for _, encName := range []string{"base32", "base64", "base32768"} {
		enc, err := NewNameEncoding(encName)
		if err != nil {
			t.Fatalf("NewNameEncoding(%q): %v", encName, err)
		}
		c, err := newCipher(NameEncryptionStandard, "", "", true, enc)
		if err != nil {
			t.Fatalf("newCipher(standard/%s): %v", encName, err)
		}
		out["standard-"+encName] = c
	}
	obf, err := newCipher(NameEncryptionObfuscated, "", "", true, nil)
	if err != nil {
		t.Fatalf("newCipher(obfuscated): %v", err)
	}
	out["obfuscated"] = obf
	off, err := newCipher(NameEncryptionOff, "", "", true, nil)
	if err != nil {
		t.Fatalf("newCipher(off): %v", err)
	}
	out["off"] = off
	return out
}

// drawPlainName generates a valid relative path name: 1-3 segments, each
// starting with a letter and otherwise letters/digits/dot/underscore.
// (No '-' anywhere, so no segment can look like an rclone --versions
// suffix, which EncryptFileName deliberately leaves in plain text.)
func drawPlainName(t *rapid.T) string {
	seg := func(label string) string {
		first := rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")).Draw(t, label+"-first")
		rest := rapid.StringOfN(rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._")), 0, 16, -1).Draw(t, label+"-rest")
		return string(first) + rest
	}
	n := rapid.IntRange(1, 3).Draw(t, "segments")
	name := seg("seg0")
	for i := 1; i < n; i++ {
		name += "/" + seg("segN")
	}
	return name
}

// TestPBTCipherDecryptNeverPanics: DecryptFileName/DecryptDirName run on
// names read back from real remotes, which may be corrupt or foreign;
// they must return an error, never panic.
func TestPBTCipherDecryptNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ciphers := pbtCiphers(t)
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune("abc012./=!-"))),
		).Draw(t, "s")
		for _, c := range ciphers {
			_, _ = c.DecryptFileName(s)
			_, _ = c.DecryptDirName(s)
		}
	})
}

// TestPBTCipherEncryptDecryptRoundTrip: for every name-encryption mode,
// decrypting an encrypted file or directory name returns the original.
// This is the core contract the crypt backend's listings rely on.
func TestPBTCipherEncryptDecryptRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ciphers := pbtCiphers(t)
		name := drawPlainName(t)
		for label, c := range ciphers {
			if got, err := c.DecryptFileName(c.EncryptFileName(name)); err != nil || got != name {
				t.Fatalf("%s: DecryptFileName(EncryptFileName(%q)) = %q, %v", label, name, got, err)
			}
			if got, err := c.DecryptDirName(c.EncryptDirName(name)); err != nil || got != name {
				t.Fatalf("%s: DecryptDirName(EncryptDirName(%q)) = %q, %v", label, name, got, err)
			}
		}
		// With directory-name encryption off, dir names pass through
		// untouched and file names must still round-trip.
		enc, err := NewNameEncoding("base32")
		if err != nil {
			t.Fatalf("NewNameEncoding: %v", err)
		}
		c, err := newCipher(NameEncryptionStandard, "", "", false, enc)
		if err != nil {
			t.Fatalf("newCipher(no dir encrypt): %v", err)
		}
		if got, err := c.DecryptFileName(c.EncryptFileName(name)); err != nil || got != name {
			t.Fatalf("no-dir-encrypt: DecryptFileName(EncryptFileName(%q)) = %q, %v", name, got, err)
		}
		if got, err := c.DecryptDirName(c.EncryptDirName(name)); err != nil || got != name {
			t.Fatalf("no-dir-encrypt: DecryptDirName(EncryptDirName(%q)) = %q, %v", name, got, err)
		}
	})
}

// TestPBTObfuscateUnicodeRoundTrip: obfuscation rotates non-ASCII runes
// by range (backend/crypt/cipher.go obfuscateSegment); any rune it
// rotates must rotate back, and runes it cannot rotate are quoted.
func TestPBTObfuscateUnicodeRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		c, err := newCipher(NameEncryptionObfuscated, "", "", true, nil)
		if err != nil {
			t.Fatalf("newCipher(obfuscated): %v", err)
		}
		seg := rapid.StringOfN(rapid.RuneFrom([]rune("abZ09.!éàÿĀāαΩ😀ﬀ")), 1, 12, -1).Draw(t, "seg")
		if got, err := c.DecryptFileName(c.EncryptFileName(seg)); err != nil || got != seg {
			t.Fatalf("obfuscated round trip of %q = %q, %v", seg, got, err)
		}
	})
}
