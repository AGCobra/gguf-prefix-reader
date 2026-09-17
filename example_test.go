package ggufprefix_test

import (
	"bytes"
	"fmt"

	ggufprefix "github.com/AGCobra/gguf-prefix-reader"
)

func ExampleReadLittleEndianV3() {
	// This is a prefix, not a complete GGUF file.
	data := []byte{
		'G', 'G', 'U', 'F', 3, 0, 0, 0,
		2, 0, 0, 0, 0, 0, 0, 0,
		1, 0, 0, 0, 0, 0, 0, 0,
	}
	prefix, err := ggufprefix.ReadLittleEndianV3(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	fmt.Printf("version=%d tensors=%d metadata=%d\n",
		prefix.Version, prefix.TensorCount, prefix.MetadataKVCount)
	// Output: version=3 tensors=2 metadata=1
}
