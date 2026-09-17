GGUF Prefix Reader
==================

A small Go package that reads only the first 24 bytes of a little-endian
GGUF version 3 stream: magic, version, tensor count, and metadata-pair count.
The implementation uses only the Go standard library and requires Go 1.22+.

Import:

    import ggufprefix "github.com/AGCobra/gguf-prefix-reader"

Usage with an already-open io.Reader named r:

    prefix, err := ggufprefix.ReadLittleEndianV3(r)
    if err != nil {
        return err
    }
    fmt.Println(prefix.Version, prefix.TensorCount, prefix.MetadataKVCount)

Run the self-contained example and checks with:

    go test ./...

The external-package example in example_test.go constructs a 24-byte prefix
and prints "version=3 tensors=2 metadata=1".

Precise limits
--------------

ReadLittleEndianV3 uses io.ReadFull to read 24 bytes. It checks the literal
magic bytes GGUF and requires version 3 encoded as a little-endian uint32.
It then decodes the two counts as little-endian uint64 values. It does not
guess or detect byte order, and it does not support big-endian GGUF variants
or other format versions. Know the stream's byte order before using it.

Only the prefix is parsed. Metadata, tensor descriptions, tensor data,
offsets, count consistency, and total file completeness are not validated.
Even a stream consisting solely of a valid 24-byte prefix can succeed.
Zero and maximum uint64 counts are accepted; the function allocates no
storage based on either count.

The function requests no bytes after byte 24 from the supplied reader.
A buffered reader may independently read ahead from its underlying source.
On a short input, fewer than 24 bytes are consumed and the read error is
wrapped; errors.Is can check io.EOF, io.ErrUnexpectedEOF, or a source error.
As with io.ReadFull, an error returned alongside a complete 24-byte read is
ignored. Format errors are reported only after the 24-byte read completes.
Every error returns a zero Prefix.

Format reference: ggml-org/ggml docs/gguf.md, reviewed at blob
21c5e8a2e5631fc9175788f5469186d1361c42e6.
https://github.com/ggml-org/ggml/blob/master/docs/gguf.md

License: MIT; see LICENSE. This package is an independent implementation.
