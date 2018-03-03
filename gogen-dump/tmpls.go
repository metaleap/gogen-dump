package main

import (
	"unsafe"
)

func foo() {
	var data []byte

	l := uint64(len(data))
	b := (*[8]byte)(unsafe.Pointer(&l))[:]
	println(b)

	var pi64 float64 = 3.14
	var b64 = (*[8]byte)(unsafe.Pointer(&pi64))[:]
	var pi32 float32 = 3.14
	var b32 = (*[4]byte)(unsafe.Pointer(&pi32))[:]
	println(b64)
	println(b32)
	p32 := (*float32)(unsafe.Pointer(&b32[0]))
	p64 := (*float64)(unsafe.Pointer(&b64[0]))
	println(*p32)
	println(*p64)
}

type tmplDotFile struct {
	ProgHint string
	PName    string
	Types    []tmplDotType
}

type tmplDotType struct {
	TName    string
	Fields   []tmplDotField
	HasWData bool
}

type tmplDotField struct {
	FName string
	TmplW string
	TmplR string
}

const tmplPkg = `package {{.PName}}

// Code generated by {{.ProgHint}} — DO NOT EDIT.

import (
	"bytes"
	"io"
	"unsafe"
)

{{range .Types}}
func (me *{{.TName}}) writeTo(buf *bytes.Buffer) (err error) {
	{{if .HasWData}}var data bytes.Buffer{{end}}
	{{range .Fields}}
	{{.TmplW}}
	{{end}}
	return
}

func (me *{{.TName}}) WriteTo(w io.Writer) (int64, error) {
	if data, err := me.MarshalBinary(); err == nil {
		var n int
		n, err = w.Write(data)
		return int64(n), err
	} else {
		return 0, err
	}
}

func (me *{{.TName}}) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *{{.TName}}) UnmarshalBinary(data []byte) (err error) {
	var i int
	{{range .Fields}}
	{{.TmplR}}
	{{end}}
	if i > 0 {}
	return
}
{{end}}
`
