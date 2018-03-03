package main

// Code generated by github.com/go-leap/gen/gogen-dump — DO NOT EDIT.

import (
	"bytes"
	"io"
	"unsafe"
)


func (me *testStruct) writeTo(buf *bytes.Buffer) (err error) {
	var data bytes.Buffer
	
	me.embName.writeTo(&data) ; l_embName := uint64(data.Len()) ; buf.Write((*[8]byte)(unsafe.Pointer(&l_embName))[:]) ; data.WriteTo(buf)
	
	if me.Deleted { buf.WriteByte(1) } else { buf.WriteByte(0) }
	
	//ident:complex128
	
	//ident:float64
	
	buf.WriteByte(me.Age)
	
	//ident:rune
	
	return
}

func (me *testStruct) WriteTo(w io.Writer) (int64, error) {
	if data, err := me.MarshalBinary(); err == nil {
		var n int
		n, err = w.Write(data)
		return int64(n), err
	} else {
		return 0, err
	}
}

func (me *testStruct) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *testStruct) UnmarshalBinary(data []byte) (err error) {
	var i int
	
	l_embName := int(*((*uint64)(unsafe.Pointer(&data[i])))) ; i += 8 ; me.embName.UnmarshalBinary(data[i : i+l_embName]) ; i += l_embName
	
	me.Deleted = (data[i] == 1) ; i++
	
	
	
	
	
	me.Age = data[i] ; i++
	
	
	
	if i > 0 {}
	return
}

func (me *embName) writeTo(buf *bytes.Buffer) (err error) {
	
	
	l_FirstName := uint64(len(me.FirstName)) ; buf.Write((*[8]byte)(unsafe.Pointer(&l_FirstName))[:]) ; buf.WriteString(me.FirstName)
	
	//no-ident:*ast.ArrayType
	
	l_LastName := uint64(len(me.LastName)) ; buf.Write((*[8]byte)(unsafe.Pointer(&l_LastName))[:]) ; buf.WriteString(me.LastName)
	
	return
}

func (me *embName) WriteTo(w io.Writer) (int64, error) {
	if data, err := me.MarshalBinary(); err == nil {
		var n int
		n, err = w.Write(data)
		return int64(n), err
	} else {
		return 0, err
	}
}

func (me *embName) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *embName) UnmarshalBinary(data []byte) (err error) {
	var i int
	
	
	
	
	
	
	
	if i > 0 {}
	return
}

