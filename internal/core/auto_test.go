package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAutoModeRoundTrips(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"empty", []byte{}, "utf8"},
		{"ascii", []byte("File splitter regression test.\r\n"), "utf8"},
		{"unicode-bom", []byte("\xef\xbb\xbfEnglish, سلام, 中文, 😀\r\n"), "utf8"},
		{"replacement-character-is-valid", []byte("Literal replacement character: \ufffd\n"), "utf8"},
		{"nul-only", []byte{0}, "base64"},
		{"embedded-nul", []byte("Good text\x00more text\r\n"), "base64"},
		{"utf16le-bom", []byte{0xff, 0xfe, 'A', 0, 'B', 0, 13, 0, 10, 0}, "base64"},
		{"utf16le-no-bom", []byte{'A', 0, 'B', 0, 'C', 0}, "base64"},
		{"utf16be-bom", []byte{0xfe, 0xff, 0, 'A', 0, 'B', 0, 10}, "base64"},
		{"utf16be-no-bom", []byte{0, 'A', 0, 'B', 0, 'C'}, "base64"},
		{"utf32le-bom", []byte{0xff, 0xfe, 0, 0, 'A', 0, 0, 0}, "base64"},
		{"utf32be-bom", []byte{0, 0, 0xfe, 0xff, 0, 0, 0, 'A'}, "base64"},
		{"windows1252", []byte{'c', 'a', 'f', 0xe9, ' ', 0x96, ' ', 0x80}, "base64"},
		{"overlong-utf8", []byte{0xc0, 0xaf}, "base64"},
		{"incomplete-last-rune", []byte{'A', 0xe2, 0x82}, "base64"},
		{"invalid-at-text-boundary", bytes.Repeat([]byte{0x80}, 200), "base64"},
		{"arbitrary-binary", payload(4099), "base64"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, src, r := fixture(t, c.data, "auto", HeaderReserve+101)
			if r.Encoding != c.want {
				t.Fatalf("encoding=%s, want %s", r.Encoding, c.want)
			}
			parts, e := Inspect(r.Folder)
			if e != nil {
				t.Fatal(e)
			}
			for _, p := range parts {
				if p.Encoding != c.want || p.Version != 1 || p.TransportSize > HeaderReserve+101 {
					t.Fatal("wrong encoding, changed format or oversized part")
				}
				raw, e := os.ReadFile(p.Path)
				if e != nil || !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
					t.Fatalf("TXT transport is not NUL-free UTF-8: %v", e)
				}
			}
			dest := filepath.Join(dir, "reconstructed.bin")
			joined, e := Join(context.Background(), r.Folder, dest, nil)
			if e != nil {
				t.Fatal(e)
			}
			got, e := os.ReadFile(dest)
			if e != nil || !bytes.Equal(got, c.data) {
				t.Fatalf("reconstructed bytes differ: %v", e)
			}
			source, e := os.ReadFile(src)
			if e != nil || !bytes.Equal(source, c.data) {
				t.Fatalf("source bytes changed: %v", e)
			}
			h := sha256.Sum256(c.data)
			if r.SHA256 != hex.EncodeToString(h[:]) || joined.SHA256 != r.SHA256 {
				t.Fatal("whole-file SHA-256 differs")
			}
		})
	}
}

func TestAutoValidatesBeyondInitialSample(t *testing.T) {
	for _, suffix := range [][]byte{{0, 'X'}, {0xff, 'X'}, {0xe2, 0x82}} {
		data := append(bytes.Repeat([]byte("Valid ASCII line\r\n"), 180000), suffix...)
		dir, _, r := fixture(t, data, "auto", 1_100_000)
		if r.Encoding != "base64" {
			t.Fatal("late incompatible bytes were missed")
		}
		if _, e := Join(context.Background(), r.Folder, filepath.Join(dir, "joined"), nil); e != nil {
			t.Fatal(e)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "joined"))
		if !bytes.Equal(got, data) {
			t.Fatal("retry did not rewind source or reset plan")
		}
	}
}

func TestAutoUTF8AcrossReadAndPartBoundaries(t *testing.T) {
	// First emoji crosses the 1 MiB read boundary. Later characters cross
	// candidate part boundaries, and every output must remain valid UTF-8.
	data := append(bytes.Repeat([]byte("A"), blockSize-1), []byte(strings.Repeat("😀سلام", 180000))...)
	dir, _, r := fixture(t, data, "auto", 1_100_003)
	if r.Encoding != "utf8" {
		t.Fatal("valid multibyte text was unnecessarily encoded")
	}
	parts, _ := Inspect(r.Folder)
	for _, p := range parts {
		b, _ := os.ReadFile(p.Path)
		if !utf8.Valid(b[p.PayloadStart:]) {
			t.Fatal("part splits a UTF-8 character")
		}
	}
	if _, e := Join(context.Background(), r.Folder, filepath.Join(dir, "joined"), nil); e != nil {
		t.Fatal(e)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "joined"))
	if !bytes.Equal(got, data) {
		t.Fatal("Unicode bytes changed")
	}
}

func TestReadableCompatibilityErrorsAreTyped(t *testing.T) {
	for _, data := range [][]byte{[]byte("A\x00B"), {0xff}, {0xe2, 0x82}, bytes.Repeat([]byte{0x80}, 200)} {
		dir := t.TempDir()
		src := filepath.Join(dir, "strict.txt")
		if e := os.WriteFile(src, data, 0600); e != nil {
			t.Fatal(e)
		}
		_, e := Split(context.Background(), Options{src, dir, "utf8", HeaderReserve + 101}, nil)
		if !errors.Is(e, ErrNotReadableUTF8) {
			t.Fatalf("expected typed compatibility error: %v", e)
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 1 {
			t.Fatal("strict mode published output for incompatible text")
		}
	}
}

func TestAutoDoesNotFallbackOnUnrelatedFailures(t *testing.T) {
	for _, kind := range []string{"cancelled", "missing-source", "invalid-output", "invalid-size", "io-error"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "source.txt")
			if e := os.WriteFile(src, bytes.Repeat([]byte("ASCII"), 1000), 0600); e != nil {
				t.Fatal(e)
			}
			o := Options{src, dir, "auto", HeaderReserve + 101}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switched := false
			n := func(p Progress) {
				if strings.Contains(p.Message, "switching to Encoded") {
					switched = true
				}
				if kind == "io-error" && p.Phase == "Scan & hash" && p.Message != "" {
					// A genuine read failure must not be hidden by Auto fallback.
					if e := os.Truncate(src, 0); e != nil {
						t.Fatal(e)
					}
				}
			}
			switch kind {
			case "cancelled":
				cancel()
			case "missing-source":
				o.Source = filepath.Join(dir, "missing")
			case "invalid-output":
				o.OutputParent = filepath.Join(dir, "missing")
			case "invalid-size":
				o.MaxPartBytes = 1
			}
			_, e := Split(ctx, o, n)
			if e == nil || switched || errors.Is(e, ErrNotReadableUTF8) {
				t.Fatalf("unrelated failure swallowed/misclassified: %v, switched=%v", e, switched)
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 {
				t.Fatal("failed split left output files")
			}
		})
	}
}

func TestAutoCancellationDuringFallback(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	if e := os.WriteFile(src, []byte("A\x00B"), 0600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, e := Split(ctx, Options{src, dir, "auto", HeaderReserve + 101}, func(p Progress) {
		if strings.Contains(p.Message, "switching to Encoded") {
			cancel()
		}
	})
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("expected cancellation: %v", e)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("cancelled fallback published output")
	}
}

func TestAutoReportsActualMode(t *testing.T) {
	for _, data := range [][]byte{[]byte("Hello"), []byte("Hello\x00")} {
		dir := t.TempDir()
		src := filepath.Join(dir, "source.txt")
		os.WriteFile(src, data, 0600)
		var messages []string
		r, e := Split(context.Background(), Options{src, dir, "auto", 20_000_000}, func(p Progress) {
			messages = append(messages, p.Message)
		})
		if e != nil {
			t.Fatal(e)
		}
		text := strings.Join(messages, "\n")
		if r.Encoding == "base64" && !strings.Contains(text, "switching to Encoded text") {
			t.Fatal("silent Base64 fallback")
		}
		if r.Encoding == "utf8" && !strings.Contains(text, "complete source passed UTF-8 validation") {
			t.Fatal("readable selection not reported")
		}
	}
}
