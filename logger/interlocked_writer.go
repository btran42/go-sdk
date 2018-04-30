package logger

import (
	"bufio"
	"io"
	"sync"
)

// NewInterlockedWriter returns a new interlocked writer.
func NewInterlockedWriter(output io.Writer) io.Writer {
	if typed, isTyped := output.(*InterlockedWriter); isTyped {
		return typed
	}

	return &InterlockedWriter{
		output:   output,
		buffered: bufio.NewWriter(output),
	}
}

// InterlockedWriter is a writer that serializes access to the Write() method.
type InterlockedWriter struct {
	sync.Mutex
	output   io.Writer
	buffered *bufio.Writer
}

// Write writes the given bytes to the inner writer.
func (iw *InterlockedWriter) Write(buffer []byte) (count int, err error) {
	iw.Lock()
	count, err = iw.buffered.Write(buffer)
	iw.buffered.Flush()
	iw.Unlock()
	return
}

// Flush flushes the buffer.
func (iw *InterlockedWriter) Flush() (err error) {
	iw.Lock()
	err = iw.buffered.Flush()
	iw.Unlock()
	return
}

// Close closes any outputs that are io.WriteCloser's.
func (iw *InterlockedWriter) Close() (err error) {
	iw.Lock()
	if err = iw.buffered.Flush(); err != nil {
		return
	}
	if typed, isTyped := iw.output.(io.WriteCloser); isTyped {
		err = typed.Close()
	}
	iw.Unlock()
	return
}
