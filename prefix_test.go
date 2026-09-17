package ggufprefix_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	ggufprefix "github.com/AGCobra/gguf-prefix-reader"
)

func distinctivePrefix() []byte {
	return []byte{
		'G', 'G', 'U', 'F', 3, 0, 0, 0,
		1, 2, 3, 4, 5, 6, 7, 8,
		0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0,
	}
}

func TestDistinctiveCounts(t *testing.T) {
	got, err := ggufprefix.ReadLittleEndianV3(bytes.NewReader(distinctivePrefix()))
	want := ggufprefix.Prefix{
		Version: 3, TensorCount: 0x0807060504030201, MetadataKVCount: 0xf0e0d0c0b0a09080,
	}
	if err != nil || got != want {
		t.Fatalf("got %+v, %v; want %+v, nil", got, err, want)
	}
}

func TestCountsAreNotValidated(t *testing.T) {
	for _, count := range []uint64{0, ^uint64(0)} {
		data := distinctivePrefix()
		binary.LittleEndian.PutUint64(data[8:16], count)
		binary.LittleEndian.PutUint64(data[16:24], count)
		got, err := ggufprefix.ReadLittleEndianV3(bytes.NewReader(data))
		if err != nil || got.TensorCount != count || got.MetadataKVCount != count {
			t.Fatalf("count %d: got %+v, %v", count, got, err)
		}
	}
}

type chunkReader struct{ r io.Reader }

func (r chunkReader) Read(p []byte) (int, error) {
	if len(p) > 3 {
		p = p[:3]
	}
	return r.r.Read(p)
}

func TestShortReads(t *testing.T) {
	data := distinctivePrefix()
	got, err := ggufprefix.ReadLittleEndianV3(chunkReader{bytes.NewReader(data)})
	if err != nil || got.TensorCount != 0x0807060504030201 || got.MetadataKVCount != 0xf0e0d0c0b0a09080 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestTruncation(t *testing.T) {
	data := distinctivePrefix()
	for n := 0; n < len(data); n++ {
		got, err := ggufprefix.ReadLittleEndianV3(bytes.NewReader(data[:n]))
		wantErr := io.ErrUnexpectedEOF
		if n == 0 {
			wantErr = io.EOF
		}
		if !errors.Is(err, wantErr) || got != (ggufprefix.Prefix{}) {
			t.Fatalf("length %d: got %+v, %v; want zero Prefix and %v", n, got, err, wantErr)
		}
	}
}

type failingReader struct {
	data []byte
	err  error
}

func (r *failingReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, r.err
}

func TestReadErrorPropagation(t *testing.T) {
	wantErr := errors.New("source failed")
	for _, n := range []int{0, 7, 23} {
		r := &failingReader{data: distinctivePrefix()[:n], err: wantErr}
		got, err := ggufprefix.ReadLittleEndianV3(r)
		if !errors.Is(err, wantErr) || got != (ggufprefix.Prefix{}) {
			t.Fatalf("length %d: got %+v, %v; want zero Prefix and wrapped source error", n, got, err)
		}
		if err.Error() == wantErr.Error() {
			t.Fatalf("read error lacks prefix context: %v", err)
		}
	}
}

func TestCompleteReadWithError(t *testing.T) {
	// io.ReadFull ignores a source error if the requested bytes were all read.
	r := &failingReader{data: distinctivePrefix(), err: errors.New("error after prefix")}
	got, err := ggufprefix.ReadLittleEndianV3(r)
	if err != nil || got.Version != 3 || got.TensorCount != 0x0807060504030201 {
		t.Fatalf("got %+v, %v; want a decoded prefix and nil", got, err)
	}
}

func TestRejectsMagicAndVersion(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func([]byte)
		message string
	}{
		{"bad magic", func(b []byte) { copy(b[:4], "FGUG") }, "magic"},
		{"version zero", func(b []byte) { binary.LittleEndian.PutUint32(b[4:8], 0) }, "version"},
		{"version two", func(b []byte) { binary.LittleEndian.PutUint32(b[4:8], 2) }, "version"},
		{"version four", func(b []byte) { binary.LittleEndian.PutUint32(b[4:8], 4) }, "version"},
		{"big endian version three", func(b []byte) { binary.BigEndian.PutUint32(b[4:8], 3) }, "version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := distinctivePrefix()
			tc.mutate(data)
			got, err := ggufprefix.ReadLittleEndianV3(bytes.NewReader(data))
			if err == nil || !strings.Contains(err.Error(), tc.message) || got != (ggufprefix.Prefix{}) {
				t.Fatalf("got %+v, %v; want zero Prefix and %s error", got, err, tc.message)
			}
		})
	}
}

type prefixBoundaryReader struct {
	r         *bytes.Reader
	remaining int
	requests  []int
}

func (r *prefixBoundaryReader) Read(p []byte) (int, error) {
	r.requests = append(r.requests, len(p))
	if len(p) > r.remaining {
		return 0, errors.New("read requested beyond prefix boundary")
	}
	n, err := r.r.Read(p)
	r.remaining -= n
	return n, err
}

func TestDoesNotReadBeyondPrefix(t *testing.T) {
	for _, valid := range []bool{true, false} {
		data := distinctivePrefix()
		if !valid {
			data[0] = 'X'
		}
		data = append(data, []byte("untouched payload")...)
		r := &prefixBoundaryReader{r: bytes.NewReader(data), remaining: 24}
		_, err := ggufprefix.ReadLittleEndianV3(r)
		if (err == nil) != valid {
			t.Fatalf("valid=%v: unexpected error %v", valid, err)
		}
		if len(r.requests) != 1 || r.requests[0] != 24 || r.remaining != 0 {
			t.Fatalf("valid=%v: requests=%v, remaining=%d", valid, r.requests, r.remaining)
		}
		rest, err := io.ReadAll(r.r)
		if err != nil || string(rest) != "untouched payload" {
			t.Fatalf("valid=%v: remaining bytes %q, %v", valid, rest, err)
		}
	}
}
