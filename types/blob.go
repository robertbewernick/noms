package types

import (
	"io"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/attic-labs/buzhash"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/d"
)

const (
	// 12 bits leads to an average size of 4k
	// 13 bits leads to an average size of 8k
	// 14 bits leads to an average size of 16k
	blobPattern = uint32(1<<13 - 1)

	// The window size to use for computing the rolling hash.
	blobWindowSize = 64
)

var typeForBlob = MakePrimitiveType(BlobKind)

type Blob interface {
	Value
	Len() uint64
	// BUG 155 - Should provide Write... Maybe even have Blob implement ReadWriteSeeker
	Reader() io.ReadSeeker
}

func NewEmptyBlob() Blob {
	return newBlobLeaf([]byte{})
}

func NewMemoryBlob(r io.Reader) Blob {
	return NewBlob(r, chunks.NewMemoryStore())
}

func newBlobLeafBoundaryChecker() boundaryChecker {
	return newBuzHashBoundaryChecker(blobWindowSize, func(h *buzhash.BuzHash, item sequenceItem) bool {
		b := item.(byte)
		return h.HashByte(b)&blobPattern == blobPattern
	})
}

func newBlobLeafChunkFn(cs chunks.ChunkStore) makeChunkFn {
	return func(items []sequenceItem) (sequenceItem, Value) {
		buff := make([]byte, len(items))

		for i, v := range items {
			buff[i] = v.(byte)
		}

		leaf := newBlobLeaf(buff)
		ref := WriteValue(leaf, cs)
		return metaTuple{ref, Uint64(uint64(len(buff)))}, leaf
	}
}

func NewBlob(r io.Reader, cs chunks.ChunkStore) Blob {
	seq := newEmptySequenceChunker(newBlobLeafChunkFn(cs), newMetaSequenceChunkFn(typeForBlob, cs), newBlobLeafBoundaryChecker(), newMetaSequenceBoundaryChecker)
	buf := []byte{0}
	for {
		n, err := r.Read(buf)
		d.Chk.True(n <= 1)
		if n == 1 {
			seq.Append(buf[0])
		}
		if err != nil {
			d.Chk.Equal(io.EOF, err)
			break
		}
	}
	return seq.Done().(Blob)
}
