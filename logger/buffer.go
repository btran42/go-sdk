package logger

import "unicode/utf8"

// Buffer is an interface.
type Buffer interface {
	WriteRune(rune)
	WriteString(string)
	Write([]byte) (int, error)
	Len() int
	Cap() int
	Reset()
	String() string
	Bytes() []byte
}

// NewBuffer returns a new buffer.
func NewBuffer(size int) *ArrayBuffer {
	return &ArrayBuffer{bs: make([]byte, 0, size)}
}

// ArrayBuffer is a thin wrapper around a byte slice.
type ArrayBuffer struct {
	bs []byte
}

// Len returns the length of the underlying byte slice.
func (b *ArrayBuffer) Len() int {
	return len(b.bs)
}

// Cap returns the capacity of the underlying byte slice.
func (b *ArrayBuffer) Cap() int {
	return cap(b.bs)
}

// Bytes returns a mutable reference to the underlying byte slice.
func (b *ArrayBuffer) Bytes() []byte {
	return b.bs
}

// String returns a string copy of the underlying byte slice.
func (b *ArrayBuffer) String() string {
	return string(b.bs)
}

// Reset resets the underlying byte slice. Subsequent writes re-use the slice's
// backing array.
func (b *ArrayBuffer) Reset() {
	b.bs = b.bs[:0]
}

// WriteString writes a string to the Buffer.
func (b *ArrayBuffer) WriteString(s string) {
	b.bs = append(b.bs, s...)
}

// WriteRune writes a rune.
func (b *ArrayBuffer) WriteRune(r rune) {
	if r < utf8.RuneSelf {
		b.bs = append(b.bs, byte(r))
		return
	}
	grow := make([]byte, utf8.RuneLen(r))
	utf8.EncodeRune(grow, r)
	b.bs = append(b.bs, grow...)
}

// Write implements io.Writer.
func (b *ArrayBuffer) Write(bs []byte) (int, error) {
	b.bs = append(b.bs, bs...)
	return len(bs), nil
}

// Sync is a no op.
func (b *ArrayBuffer) Sync() error {
	return nil
}
