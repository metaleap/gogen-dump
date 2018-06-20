package main

import (
	"io"
)

// DO keep manually in sync with the copy in `gen-tmpl.go`'s `const tmplSrc`!
type writeBuf struct{ b []byte }

func writeBuffer(b []byte) *writeBuf {
	return &writeBuf{b: b}
}

func (this *writeBuf) copyTo(to *writeBuf) {
	to.write(this.b)
}

func (this *writeBuf) writeByte(b byte) {
	l, c := len(this.b), cap(this.b)
	if l == c {
		old := this.b
		this.b = make([]byte, l+1, l+l+128) // the constant extra padding: if we're tiny (~0), it helps much; if we're large (MBs), it hurts none
		copy(this.b[:l], old)
	} else {
		this.b = this.b[:l+1]
	}
	this.b[l] = b
}

func (this *writeBuf) write(b []byte) {
	l, c, n := len(this.b), cap(this.b), len(b)
	if ln := l + n; ln > c {
		old := this.b
		this.b = make([]byte, ln, ln+ln+128)
		copy(this.b[:l], old)
	} else {
		this.b = this.b[:ln]
	}
	copy(this.b[l:], b)
}

func (this *writeBuf) writeString(b string) {
	l, c, n := len(this.b), cap(this.b), len(b)
	if ln := l + n; ln > c {
		old := this.b
		this.b = make([]byte, ln, ln+ln+128)
		copy(this.b[:l], old)
	} else {
		this.b = this.b[:ln]
	}
	copy(this.b[l:], b)
}

func (this *writeBuf) writeTo(w io.Writer) (int64, error) {
	n, err := w.Write(this.b)
	return int64(n), err
}
