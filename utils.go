package main

import (
	"io"
)

type bytesBuffer struct{ b []byte }

func (me *bytesBuffer) bytes() []byte {
	return me.b
}

func (me *bytesBuffer) copyTo(to *bytesBuffer) {
	to.write(me.b)
}

func (me *bytesBuffer) reset() {
	me.b = me.b[:0]
}

func (me *bytesBuffer) writeByte(b byte) {
	l, c := len(me.b), cap(me.b)
	if l == c {
		old := me.b
		me.b = make([]byte, l+1, l+l)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:l+1]
	}
	me.b[l] = b
}

func (me *bytesBuffer) write(b []byte) {
	l, c, n := len(me.b), cap(me.b), len(b)
	if ln := l + n; ln > c {
		old := me.b
		me.b = make([]byte, ln, ln+ln)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:ln]
	}
	copy(me.b[l:], b)
}

func (me *bytesBuffer) writeString(b string) {
	l, c, n := len(me.b), cap(me.b), len(b)
	if ln := l + n; ln > c {
		old := me.b
		me.b = make([]byte, ln, ln+ln)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:ln]
	}
	copy(me.b[l:], b)
}

func (me *bytesBuffer) writeTo(w io.Writer) (int64, error) {
	n, err := w.Write(me.b)
	return int64(n), err
}
