package main

import (
	"io"
)

type writeBuf struct{ b []byte }

func writeBuffer(b []byte) *writeBuf {
	return &writeBuf{b: b}
}

func (me *writeBuf) copyTo(to *writeBuf) {
	to.write(me.b)
}

func (me *writeBuf) writeByte(b byte) {
	l, c := len(me.b), cap(me.b)
	if l == c {
		old := me.b
		me.b = make([]byte, l+1, l+l+128)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:l+1]
	}
	me.b[l] = b
}

func (me *writeBuf) write(b []byte) {
	l, c, n := len(me.b), cap(me.b), len(b)
	if ln := l + n; ln > c {
		old := me.b
		me.b = make([]byte, ln, ln+ln+128)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:ln]
	}
	copy(me.b[l:], b)
}

func (me *writeBuf) writeString(b string) {
	l, c, n := len(me.b), cap(me.b), len(b)
	if ln := l + n; ln > c {
		old := me.b
		me.b = make([]byte, ln, ln+ln+128)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:ln]
	}
	copy(me.b[l:], b)
}

func (me *writeBuf) writeTo(w io.Writer) (int64, error) {
	n, err := w.Write(me.b)
	return int64(n), err
}
