package utils

import (
	"strings"
	"testing"
)

func TestEncodeDecodeAccountArchive_RoundTrip(t *testing.T) {
	plain := strings.Join([]string{
		"a@example.com----p1----cid1----rt1----默认分组",
		"b@example.com----p2----cid2----rt2----营销组",
	}, "\n")

	encoded, err := EncodeAccountArchive(plain)
	if err != nil {
		t.Fatalf("EncodeAccountArchive returned error: %v", err)
	}

	if IsAccountArchive([]byte(plain)) {
		t.Fatalf("plain text should not be detected as archive")
	}

	if !IsAccountArchive(encoded) {
		t.Fatalf("encoded bytes should be detected as archive")
	}

	decoded, err := DecodeAccountArchive(encoded)
	if err != nil {
		t.Fatalf("DecodeAccountArchive returned error: %v", err)
	}

	if decoded != plain {
		t.Fatalf("decoded content mismatch\nwant: %q\n got: %q", plain, decoded)
	}
}

func TestDecodeAccountArchive_InvalidMagic(t *testing.T) {
	_, err := DecodeAccountArchive([]byte("NOT-ARCHIVE"))
	if err == nil {
		t.Fatalf("expected error for invalid magic")
	}
}

func TestDecodeAccountArchive_UnsupportedVersion(t *testing.T) {
	plain := "a@example.com----p1----cid1----rt1----默认分组"
	encoded, err := EncodeAccountArchive(plain)
	if err != nil {
		t.Fatalf("EncodeAccountArchive returned error: %v", err)
	}

	// magic(6 bytes) + version(1 byte)
	encoded[6] = 2

	_, err = DecodeAccountArchive(encoded)
	if err == nil {
		t.Fatalf("expected error for unsupported version")
	}
}

func TestDecodeAccountArchive_CorruptedPayload(t *testing.T) {
	plain := "a@example.com----p1----cid1----rt1----默认分组"
	encoded, err := EncodeAccountArchive(plain)
	if err != nil {
		t.Fatalf("EncodeAccountArchive returned error: %v", err)
	}

	// Corrupt tail bytes
	encoded[len(encoded)-1] ^= 0xFF

	_, err = DecodeAccountArchive(encoded)
	if err == nil {
		t.Fatalf("expected error for corrupted payload")
	}
}
