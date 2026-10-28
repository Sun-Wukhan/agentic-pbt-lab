// Property-based tests for backend/chunker metadata handling.
//
// Lab: Agentic PBT Lab (targets/rclone-rclone) — pristine copy.
// run.sh copies this file into the rclone checkout at
// backend/chunker/pbt_chunker_test.go (in-package, like
// chunker_internal_test.go, because unmarshalSimpleJSON and
// marshalSimpleJSON are unexported).
//
// unmarshalSimpleJSON is called with a nil metaObject: it only stores
// the object in the returned ObjectInfo (see wrapInfo in chunker.go),
// and these tests never call methods on that ObjectInfo that would
// dereference it.
package chunker

import (
	"context"
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// TestPBTChunkerMetadataNeverPanics: chunker metadata objects live on
// real remotes and can be corrupt or foreign files misidentified as
// metadata; decoding arbitrary bytes must error, never panic.
func TestPBTChunkerMetadataNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var data []byte
		switch rapid.IntRange(0, 2).Draw(t, "kind") {
		case 0:
			data = []byte(rapid.String().Draw(t, "raw"))
		case 1:
			data = rapid.SliceOf(rapid.Uint8()).Draw(t, "bytes")
		default:
			// Structured: a JSON object with the real field names but
			// values drawn from the extremes of their types.
			data = []byte(fmt.Sprintf(`{"ver":%d,"size":%d,"nchunks":%d,"md5":%q,"sha1":%q,"txn":%q}`,
				rapid.Int().Draw(t, "ver"),
				rapid.Int64().Draw(t, "size"),
				rapid.Int().Draw(t, "nchunks"),
				rapid.SampledFrom([]string{"", "0123456789abcdef0123456789abcdef", "zz", "0123"}).Draw(t, "md5"),
				rapid.SampledFrom([]string{"", "0123456789abcdef0123456789abcdef01234567", "zz"}).Draw(t, "sha1"),
				rapid.StringN(0, 8, -1).Draw(t, "txn"),
			))
		}
		info, _, err := unmarshalSimpleJSON(context.Background(), nil, data)
		if err == nil && info == nil {
			t.Fatalf("unmarshalSimpleJSON(%q) returned nil info with nil error", data)
		}
	})
}

// TestPBTChunkerMetadataRoundTrip: metadata written by marshalSimpleJSON
// must be readable back by unmarshalSimpleJSON with size and chunk
// count preserved ("never produce something you can't consume" —
// the guard inside marshalSimpleJSON states the same intent).
func TestPBTChunkerMetadataRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.Int64Range(0, 1<<62).Draw(t, "size")
		nChunks := rapid.IntRange(1, 100000).Draw(t, "nChunks")
		md5 := rapid.SampledFrom([]string{"", "0123456789abcdef0123456789abcdef"}).Draw(t, "md5")
		sha1 := rapid.SampledFrom([]string{"", "0123456789abcdef0123456789abcdef01234567"}).Draw(t, "sha1")
		xactID := rapid.SampledFrom([]string{"", "abc123"}).Draw(t, "xactID")
		data, err := marshalSimpleJSON(context.Background(), size, nChunks, md5, sha1, xactID)
		if err != nil {
			t.Fatalf("marshalSimpleJSON: %v", err)
		}
		info, madeByChunker, err := unmarshalSimpleJSON(context.Background(), nil, data)
		if err != nil {
			t.Fatalf("unmarshalSimpleJSON(marshalSimpleJSON(...)) errored: %v (data %q)", err, data)
		}
		if !madeByChunker {
			t.Fatalf("own metadata not recognised as chunker-made: %q", data)
		}
		if info.Size() != size || info.nChunks != nChunks {
			t.Fatalf("round trip gave size=%d nChunks=%d, want %d/%d (data %q)", info.Size(), info.nChunks, size, nChunks, data)
		}
	})
}
