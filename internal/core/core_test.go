package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, data []byte, enc string, cap int64) (string, string, Result) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "source_گزارش.dat")
	if e := os.WriteFile(src, data, 0600); e != nil {
		t.Fatal(e)
	}
	r, e := Split(context.Background(), Options{src, dir, enc, cap}, nil)
	if e != nil {
		t.Fatal(e)
	}
	return dir, src, r
}
func payload(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte((i*127 + i/19 + 31) % 256)
	}
	return b
}
func TestRoundTrips(t *testing.T) {
	for _, enc := range []string{"binary", "base64"} {
		for _, size := range []int{0, 1, 2, 3, 15, 16, 199, 200, 201, 997, 3000} {
			t.Run(fmt.Sprintf("%s-%d", enc, size), func(t *testing.T) {
				data := payload(size)
				dir, src, r := fixture(t, data, enc, HeaderReserve+200)
				p, e := Inspect(r.Folder)
				if e != nil {
					t.Fatal(e)
				}
				for _, piece := range p {
					if piece.TransportSize > HeaderReserve+200 {
						t.Fatal("oversized part")
					}
				}
				dest := filepath.Join(dir, "joined.dat")
				joined, e := Join(context.Background(), r.Folder, dest, nil)
				if e != nil {
					t.Fatal(e)
				}
				got, _ := os.ReadFile(dest)
				if !bytes.Equal(got, data) {
					t.Fatal("different bytes")
				}
				source, _ := os.ReadFile(src)
				if !bytes.Equal(source, data) {
					t.Fatal("source altered")
				}
				h := sha256.Sum256(data)
				if joined.SHA256 != hex.EncodeToString(h[:]) {
					t.Fatal("wrong final hash")
				}
			})
		}
	}
}
func TestReadableUnicode(t *testing.T) {
	for _, data := range [][]byte{[]byte(""), []byte("\xef\xbb\xbfHello\r\nسلام 😀 دنیا\r\n"), []byte(strings.Repeat("A😀ب", 1000)), []byte(strings.Repeat("line αβγ\r\n", 500))} {
		dir, _, r := fixture(t, data, "utf8", HeaderReserve+101)
		parts, e := Inspect(r.Folder)
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range parts {
			raw, _ := os.ReadFile(p.Path)
			v := validator{}
			if _, e = v.Write(raw[p.PayloadStart:]); e != nil || len(v.carry) > 0 {
				t.Fatalf("invalid per-part UTF-8 %v", e)
			}
		}
		dest := filepath.Join(dir, "joined.txt")
		if _, e = Join(context.Background(), r.Folder, dest, nil); e != nil {
			t.Fatal(e)
		}
		got, _ := os.ReadFile(dest)
		if !bytes.Equal(got, data) {
			t.Fatal("Unicode changed")
		}
	}
}
func TestRejectInvalidText(t *testing.T) {
	for _, data := range [][]byte{{0xff, 0xfe, 0x41, 0}, {0x41, 0, 0x42}, {0xe2, 0x82}, {0xc0, 0xaf}} {
		dir := t.TempDir()
		src := filepath.Join(dir, "bad.txt")
		os.WriteFile(src, data, 0600)
		if _, e := Split(context.Background(), Options{src, dir, "utf8", HeaderReserve + 100}, nil); e == nil {
			t.Fatal("invalid text accepted")
		}
	}
}
func TestRenamingAndUnrelatedFiles(t *testing.T) {
	dir, _, r := fixture(t, payload(1400), "base64", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	for i, x := range p {
		os.Rename(x.Path, filepath.Join(r.Folder, fmt.Sprintf("renamed-%d.weird", len(p)-i)))
	}
	os.WriteFile(filepath.Join(r.Folder, "unrelated.txt"), []byte("not a part"), 0600)
	if _, e := Verify(context.Background(), r.Folder, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := Join(context.Background(), r.Folder, filepath.Join(dir, "out"), nil); e != nil {
		t.Fatal(e)
	}
}
func TestMissing(t *testing.T) {
	_, _, r := fixture(t, payload(1500), "base64", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	os.Remove(p[2].Path)
	if _, e := Verify(context.Background(), r.Folder, nil); e == nil || !strings.Contains(e.Error(), "missing") {
		t.Fatalf("expected missing error, got %v", e)
	}
}
func TestDuplicate(t *testing.T) {
	_, _, r := fixture(t, payload(900), "binary", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	b, _ := os.ReadFile(p[0].Path)
	os.WriteFile(filepath.Join(r.Folder, "duplicate.txt"), b, 0600)
	if _, e := Inspect(r.Folder); e == nil || !strings.Contains(e.Error(), "duplicate") {
		t.Fatalf("got %v", e)
	}
}
func TestMixedSets(t *testing.T) {
	_, _, r := fixture(t, payload(900), "binary", HeaderReserve+200)
	_, _, other := fixture(t, payload(910), "binary", HeaderReserve+200)
	p, _ := Inspect(other.Folder)
	b, _ := os.ReadFile(p[0].Path)
	os.WriteFile(filepath.Join(r.Folder, "otherpart.txt"), b, 0600)
	if _, e := Inspect(r.Folder); e == nil || !strings.Contains(e.Error(), "mixed") {
		t.Fatalf("got %v", e)
	}
}
func TestCorruptionAndNoPublishedOutput(t *testing.T) {
	for _, enc := range []string{"binary", "base64", "utf8"} {
		t.Run(enc, func(t *testing.T) {
			data := []byte(strings.Repeat("Test material\n", 60))
			dir, _, r := fixture(t, data, enc, HeaderReserve+200)
			p, _ := Inspect(r.Folder)
			b, _ := os.ReadFile(p[0].Path)
			b[p[0].PayloadStart] = 'Z'
			os.WriteFile(p[0].Path, b, 0600)
			if _, e := Verify(context.Background(), r.Folder, nil); e == nil {
				t.Fatal("corruption not detected")
			}
			dest := filepath.Join(dir, "not-created")
			if _, e := Join(context.Background(), r.Folder, dest, nil); e == nil {
				t.Fatal("corrupt join accepted")
			}
			if _, e := os.Stat(dest); !os.IsNotExist(e) {
				t.Fatal("bad output published")
			}
		})
	}
}
func TestTruncatedAndTrailing(t *testing.T) {
	for _, delta := range []int{-1, 1} {
		_, _, r := fixture(t, payload(700), "base64", HeaderReserve+200)
		p, _ := Inspect(r.Folder)
		b, _ := os.ReadFile(p[0].Path)
		if delta < 0 {
			b = b[:len(b)-1]
		} else {
			b = append(b, 'x')
		}
		os.WriteFile(p[0].Path, b, 0600)
		if _, e := Inspect(r.Folder); e == nil {
			t.Fatal("length alteration not detected")
		}
	}
}
func rewritePart(t *testing.T, p Part, change func(*Metadata)) {
	t.Helper()
	b, e := os.ReadFile(p.Path)
	if e != nil {
		t.Fatal(e)
	}
	m := p.Metadata
	change(&m)
	j, _ := json.Marshal(m)
	raw := append([]byte(Magic), j...)
	raw = append(raw, '\n', '\n')
	raw = append(raw, b[p.PayloadStart:]...)
	os.WriteFile(p.Path, raw, 0600)
}
func TestUnsafeFilename(t *testing.T) {
	_, _, r := fixture(t, payload(1), "binary", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	rewritePart(t, p[0], func(m *Metadata) { m.OriginalName = "../../evil.exe" })
	if _, e := Inspect(r.Folder); e == nil {
		t.Fatal("unsafe name accepted")
	}
}
func TestMetadataMismatch(t *testing.T) {
	_, _, r := fixture(t, payload(900), "binary", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	rewritePart(t, p[1], func(m *Metadata) { m.OriginalSize++ })
	if _, e := Inspect(r.Folder); e == nil {
		t.Fatal("metadata conflict accepted")
	}
}
func TestOffsetMismatch(t *testing.T) {
	_, _, r := fixture(t, payload(900), "binary", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	rewritePart(t, p[1], func(m *Metadata) { m.Offset-- })
	if _, e := Inspect(r.Folder); e == nil {
		t.Fatal("offset conflict accepted")
	}
}
func TestWholeHashMismatch(t *testing.T) {
	_, _, r := fixture(t, payload(900), "binary", HeaderReserve+200)
	p, _ := Inspect(r.Folder)
	for _, x := range p {
		rewritePart(t, x, func(m *Metadata) { m.OriginalSHA256 = strings.Repeat("0", 64) })
	}
	if _, e := Verify(context.Background(), r.Folder, nil); e == nil {
		t.Fatal("false whole hash accepted")
	}
}
func TestNeverOverwrite(t *testing.T) {
	dir, _, r := fixture(t, payload(700), "binary", HeaderReserve+200)
	dest := filepath.Join(dir, "existing")
	os.WriteFile(dest, []byte("KEEP"), 0600)
	if _, e := Join(context.Background(), r.Folder, dest, nil); e == nil {
		t.Fatal("overwrite allowed")
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "KEEP" {
		t.Fatal("existing file altered")
	}
}
func TestCancellation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "data")
	os.WriteFile(src, payload(10000), 0600)
	ctx, cancel := context.WithCancel(context.Background())
	n := func(p Progress) {
		if p.Phase == "Write parts" {
			cancel()
		}
	}
	if _, e := Split(ctx, Options{src, dir, "binary", HeaderReserve + 200}, n); !errors.Is(e, context.Canceled) {
		t.Fatalf("got %v", e)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Fatal("incomplete output not cleaned")
	}
}
func TestCancelledJoin(t *testing.T) {
	dir, _, r := fixture(t, payload(5000), "binary", HeaderReserve+200)
	ctx, cancel := context.WithCancel(context.Background())
	dest := filepath.Join(dir, "never")
	_, e := Join(ctx, r.Folder, dest, func(p Progress) { cancel() })
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("got %v", e)
	}
	if _, e = os.Stat(dest); !os.IsNotExist(e) {
		t.Fatal("cancelled output published")
	}
}
func TestSourceModificationDetected(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	original := payload(7000)
	os.WriteFile(src, original, 0600)
	changed := false
	_, e := Split(context.Background(), Options{src, dir, "binary", HeaderReserve + 200}, func(p Progress) {
		if !changed && p.Phase == "Scan & hash" && p.Done == p.Total && p.Total > 0 {
			changed = true
			os.WriteFile(src, bytes.Repeat([]byte("X"), 7000), 0600)
		}
	})
	if e == nil {
		t.Fatal("source mutation not detected")
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Fatal("failed source-mutation split not cleaned")
	}
}
func TestStreamingLargeFile(t *testing.T) {
	data := bytes.Repeat([]byte("abcdef0123456789\n"), 2_000_000)
	dir, _, r := fixture(t, data, "base64", 20_000_000)
	if r.Parts < 2 {
		t.Fatal("test too small")
	}
	dest := filepath.Join(dir, "joined")
	if _, e := Join(context.Background(), r.Folder, dest, nil); e != nil {
		t.Fatal(e)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, data) {
		t.Fatal("large mismatch")
	}
}
