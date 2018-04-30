package logger

import (
	"bytes"
	"sync"
)

// NewBufferPool returns a new BufferPool.
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		Pool: &sync.Pool{New: func() Any {
			return bytes.NewBuffer(make([]byte, 0, bufferSize))
		}},
	}
}

// BufferPool is a sync.Pool of Buffer.
type BufferPool struct {
	*sync.Pool
}

// Get returns a pooled Buffer instance.
func (bp *BufferPool) Get() *bytes.Buffer {
	b := bp.Pool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

// Put returns the pooled instance.
func (bp *BufferPool) Put(b *bytes.Buffer) {
	bp.Pool.Put(b)
}
