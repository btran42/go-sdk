package logger

import (
	"io"
	"sync"
)

const (
	// DefaultWriteHeadCount is the default number of write heads.
	DefaultWriteHeadCount = 4
)

// NewInterlockedWriter returns a new interlocked writer.
func NewInterlockedWriter(output io.Writer) io.Writer {
	return &InterlockedWriter{
		output:   output,
		syncRoot: &sync.Mutex{},
	}
}

// InterlockedWriter is a writer that serializes access to the Write() method.
type InterlockedWriter struct {
	output   io.Writer
	syncRoot *sync.Mutex
}

// Write writes the given bytes to the inner writer.
func (iw *InterlockedWriter) Write(buffer []byte) (count int, err error) {
	iw.syncRoot.Lock()
	count, err = iw.output.Write(buffer)
	iw.syncRoot.Unlock()
	return
}

// Close closes any outputs that are io.WriteCloser's.
func (iw *InterlockedWriter) Close() (err error) {
	iw.syncRoot.Lock()
	if typed, isTyped := iw.output.(io.WriteCloser); isTyped {
		err = typed.Close()
	}
	iw.syncRoot.Unlock()
	return
}
