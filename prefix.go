// Package ggufprefix reads the 24-byte prefix of little-endian GGUF version 3
// files. It does not read or validate metadata, tensor descriptions, or tensor
// data, and a successful read does not establish that a complete file exists.
package ggufprefix

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Prefix contains the fields before the metadata in a GGUF header.
// The counts are decoded values; their consistency with a file is not checked.
type Prefix struct {
	Version         uint32
	TensorCount     uint64
	MetadataKVCount uint64
}

// ReadLittleEndianV3 reads a 24-byte prefix with io.ReadFull, requires the literal
// magic bytes GGUF and a little-endian version value of 3, and decodes the counts
// as little-endian uint64 values. It does not detect other byte orders.
//
// It requests no bytes beyond the prefix from r, although a buffering reader may
// independently read ahead. On error, it returns a zero Prefix; read errors are
// wrapped so callers can inspect them with errors.Is.
func ReadLittleEndianV3(r io.Reader) (Prefix, error) {
	var buf [24]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return Prefix{}, fmt.Errorf("read 24-byte GGUF prefix: %w", err)
	}
	if string(buf[:4]) != "GGUF" {
		return Prefix{}, fmt.Errorf("invalid GGUF magic: %q", buf[:4])
	}
	version := binary.LittleEndian.Uint32(buf[4:8])
	if version != 3 {
		return Prefix{}, fmt.Errorf("unsupported GGUF version %d: want little-endian version 3", version)
	}
	return Prefix{
		Version:         version,
		TensorCount:     binary.LittleEndian.Uint64(buf[8:16]),
		MetadataKVCount: binary.LittleEndian.Uint64(buf[16:24]),
	}, nil
}
