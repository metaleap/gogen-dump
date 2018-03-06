package main

import (
	"bytes"
	"go/format"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-leap/str"
)

type tmplDotFile struct {
	ProgHint string
	PName    string
	Types    []*tmplDotType
	Imps     map[string]string
}

type tmplDotType struct {
	TName    string
	Fields   []*tmplDotField
	HasWData bool
	HasB0Ptr bool
	HasB1Ptr bool

	fixedsize int
}

func (me *tmplDotType) isIfaceSlice(name string) bool {
	if i := strings.Index(name, "["); i > 0 {
		name = name[:i]
		for i = range me.Fields {
			if me.Fields[i].FName == name {
				return me.Fields[i].isIfaceSlice
			}
		}
	}
	return false
}

func (me *tmplDotType) fixedSize() int {
	if me.fixedsize == 0 {
		if tsyn := tSynonyms[me.TName]; tsyn != "" {
			for _, tdt := range tdot.Types {
				if tdt.TName == tsyn {
					me.fixedsize = tdt.fixedSize()
					return me.fixedsize
				}
			}
		}
		for _, fld := range me.Fields {
			if fs := fld.fixedSize(); fs <= 0 {
				me.fixedsize = -1
				break
			} else {
				me.fixedsize += fs
			}
		}
		if me.fixedsize == 0 {
			me.fixedsize = -1
		}
	}
	return me.fixedsize
}

type tmplDotField struct {
	FName string
	TmplW string
	TmplR string

	typeIdent    string
	taggedUnion  []string
	skip         bool
	fixedsize    int
	isIfaceSlice bool
	isLast       bool
}

func (me *tmplDotField) fixedSize() int {
	if me.fixedsize == 0 {
		me.fixedsize = -1
		mult, tn := 1, me.typeIdent
		for tn[0] == '[' {
			if i := ustr.Pos(tn, "]"); i <= 1 {
				return me.fixedsize
			} else if nulen, _ := strconv.Atoi(tn[1:i]); nulen <= 0 {
				return me.fixedsize
			} else if mult, tn = mult*nulen, tn[i+1:]; tn == "" {
				return me.fixedsize
			}
		}

		tsyn := tSynonyms[tn]
		if primsize := typePrimFixedSize(tn); primsize > 0 {
			me.fixedsize = mult * primsize
		} else if primsize = typePrimFixedSize(tsyn); primsize > 0 {
			me.fixedsize = mult * primsize
		} else {
			for _, tdt := range tdot.Types {
				if tdt.TName == tsyn {
					me.fixedsize = mult * tdt.fixedSize()
					break
				} else if tdt.TName == tn {
					me.fixedsize = mult * tdt.fixedSize()
					break
				}
			}
		}
	}
	return me.fixedsize
}

const tmplPkg = `package {{.PName}}

// Code generated by {{.ProgHint}} — DO NOT EDIT.

import (
	"bytes"
	"io"
	"unsafe"
	{{range $pkgname, $pkgimppath := .Imps}}
	{{ $pkgname }} "{{$pkgimppath}}"
	{{- end}}
)

{{range .Types}}
func (me *{{.TName}}) writeTo(buf *bytes.Buffer) (err error) {
	{{if .HasWData}}var data bytes.Buffer{{end}}
	{{if .HasB0Ptr}}var b0 byte ; var b0s = (*((*[1]byte)(unsafe.Pointer(&b0))))[:] {{end}}
	{{if .HasB1Ptr}}var b1 byte = 1 ; var b1s = (*((*[1]byte)(unsafe.Pointer(&b1))))[:] {{end}}
	{{range .Fields}}
	{{.TmplW}}
	{{end}}
	return
}

func (me *{{.TName}}) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	if err := me.writeTo(&buf); err != nil {
		return 0, err
	}
	return buf.WriteTo(w)
}

func (me *{{.TName}}) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *{{.TName}}) UnmarshalBinary(data []byte) (err error) {
	var pos int
	{{range .Fields}}
	{{.TmplR}}
	{{end}}
	return
}
{{end}}
`

func genViaTmpl(file io.Writer) (err error) {
	tmpl := template.New("gen-tmpl.go")
	if _, err = tmpl.Parse(tmplPkg); err == nil {
		var buf bytes.Buffer
		if err = tmpl.Execute(&buf, &tdot); err == nil {
			src := buf.Bytes()
			if srcfmt, errfmt := format.Source(src); errfmt == nil {
				_, err = file.Write(srcfmt)
			} else {
				_, err = file.Write(src)
			}
		}
	}
	return
}
