package logger

import (
	"sync"
)

// NewBufferPool returns a new BufferPool.
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		Pool: &sync.Pool{New: func() Any {
			return &ArrayBuffer{bs: make([]byte, 0, bufferSize)}
		}},
	}
}

// BufferPool is a sync.Pool of Buffer.
type BufferPool struct {
	*sync.Pool
}

// Get returns a pooled Buffer instance.
func (bp *BufferPool) Get() Buffer {
	b := bp.Pool.Get().(Buffer)
	b.Reset()
	return b
}

// Put returns the pooled instance.
func (bp *BufferPool) Put(b Buffer) {
	bp.Pool.Put(b)
}
