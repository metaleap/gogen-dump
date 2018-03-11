package main

import (
	"bytes"
	"go/format"
	"text/template"
)

const tmplSrc = `package {{.PName}}
// Code generated by {{.ProgHint}} — DO NOT EDIT.

// This file consists solely of generated (de)serialization methods for these {{len .Structs}} struct types:
{{range .Structs}}// - {{.TName}}
{{end}}
import (
	"bytes"
	"io"
	"unsafe"
	{{range $pkgname, $pkg := .Imps}}{{if $pkg.Used}}
		{{ $pkgname }} "{{$pkg.ImportPath}}"{{end}}{{end}}
)

{{range .Structs}}
func (me *{{.TName}}) marshalTo(buf *bytes.Buffer) (err error) {
	{{if .TmplW}}
	{{.TmplW}}
	{{else}}
	{{range .Fields}}
	{{.TmplW}}
	{{end}}
	{{end}}
	return
}

// MarshalBinary implements ` + "`" + `encoding.BinaryMarshaler` + "`" + ` by serializing ` + "`" + `me` + "`" + ` into ` + "`" + `data` + "`" + `.
func (me *{{.TName}}) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, {{.InitialBufSize}}))
	if err = me.marshalTo(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

// ReadFrom implements ` + "`" + `io.ReaderFrom` + "`" + ` by deserializing from ` + "`" + `r` + "`" + ` into ` + "`" + `me` + "`" + `.
func (me *{{.TName}}) ReadFrom(r io.Reader) (n int64, err error) {
	var buf bytes.Buffer
	if n, err = buf.ReadFrom(r); err == nil {
		err = me.UnmarshalBinary(buf.Bytes())
	}
	return
}

func (me *{{.TName}}) unmarshalFrom(pos *int, data []byte) (err error) {
	p := *pos
	{{if .TmplR}}
	{{.TmplR}}
	{{else}}
	{{range .Fields}}
	{{.TmplR}}
	{{end}}
	{{end}}
	*pos = p
	return
}

// UnmarshalBinary implements ` + "`" + `encoding.BinaryUnmarshaler` + "`" + ` by deserializing from ` + "`" + `data` + "`" + ` into ` + "`" + `me` + "`" + `.
func (me *{{.TName}}) UnmarshalBinary(data []byte) (err error) {
	var pos0 int
	err = me.unmarshalFrom(&pos0, data)
	return
}

// WriteTo implements ` + "`" + `io.WriterTo` + "`" + ` by serializing ` + "`" + `me` + "`" + ` to ` + "`" + `w` + "`" + `.
func (me *{{.TName}}) WriteTo(w io.Writer) (int64, error) {
	buf := bytes.NewBuffer(make([]byte, 0, {{.InitialBufSize}}))
	if err := me.marshalTo(buf); err != nil {
		return 0, err
	}
	return buf.WriteTo(w)
}
{{end}}
`

type tmplDotFile struct {
	ProgHint string
	PName    string
	Structs  []*tmplDotStruct
	Imps     map[string]*tmplDotPkgImp

	allStructTypeDefsCollected bool
}

func (me *tmplDotFile) Len() int               { return len(me.Structs) }
func (me *tmplDotFile) Less(i int, j int) bool { return me.Structs[i].TName < me.Structs[j].TName }
func (me *tmplDotFile) Swap(i int, j int)      { me.Structs[i], me.Structs[j] = me.Structs[j], me.Structs[i] }

type tmplDotPkgImp struct {
	ImportPath string
	Used       bool
}

type tmplDotStruct struct {
	TName          string
	Fields         []*tmplDotField
	TmplR          string // only if fixedSize() > 0 && !optNoFixedSizeCode
	TmplW          string // only if fixedSize() > 0 && !optNoFixedSizeCode
	InitialBufSize string

	fixedsize   int
	sizeheuring bool
}

func (me *tmplDotStruct) fixedSize() int {
	if me.fixedsize == 0 && tdot.allStructTypeDefsCollected {
		me.fixedsize = -1 // in case of recursive type structures
		isfixedsize := true
		for _, fld := range me.Fields {
			if fs := fld.fixedSize(); fs < 0 {
				isfixedsize = false
				break
			} else if fs == 0 {
				panic("should never occur, your recent changes must have introduced a bug: " + me.TName + "." + fld.FName)
			}
		}
		if isfixedsize { // so far we really just verified fixed-size-ness but to get the correct size, need to account for alignments/paddings instead of naively summing field sizes
			if me.fixedsize = int(typeSizes.Sizeof(typeObjs[me.TName])); me.fixedsize == 0 {
				me.fixedsize = -1
			}
		}
	}
	return me.fixedsize
}

func (me *tmplDotStruct) sizeHeur(exprPref string) string {
	if me.sizeheuring {
		return optHeuristicSizeUnknowns
	}
	me.sizeheuring = true
	if fs := me.fixedSize(); fs > 0 {
		return s(fs)
	}
	var s string
	for _, tdf := range me.Fields {
		s += "+" + tdf.sizeHeur(exprPref)
	}
	me.sizeheuring = false
	return s[1:]
}

type tmplDotField struct {
	FName string
	TmplW string
	TmplR string

	typeIdent         string
	taggedUnion       []string
	skip              bool
	nextOneWasSkipped bool
	isLast            bool

	fixedsize           int
	fixedsizeExt        int
	fixedsizeExtNumSkip int
	sizeheur            string
}

func (me *tmplDotField) finalTypeIdent() (typeident string) {
	typeident = me.typeIdent
	for tsyn := typeSyns[typeident]; tsyn != ""; tsyn = typeSyns[typeident] {
		typeident = tsyn
	}
	return
}

func (me *tmplDotField) fixedSize() int {
	if me.fixedsize == 0 && tdot.allStructTypeDefsCollected {
		me.fixedsize = fixedSizeForTypeSpec(me.typeIdent)
	}
	return me.fixedsize
}

func (me *tmplDotField) sizeHeur(exprPref string) string {
	if fs := me.fixedSize(); fs > 0 {
		return s(fs)
	}
	if exprPref != "" {
		exprPref += me.FName
	}
	return typeSizeHeur(me.finalTypeIdent(), exprPref)
}

func genViaTmpl() (src []byte, err error) {
	tmpl := template.New("gen-tmpl.go")
	if _, err = tmpl.Parse(tmplSrc); err == nil {
		var buf bytes.Buffer
		if err = tmpl.Execute(&buf, &tdot); err == nil {
			src = buf.Bytes()
			if srcfmt, errfmt := format.Source(src); errfmt == nil {
				src = srcfmt
			} else {
				println("be fore-armed — the generated code could not be formatted, so it won't compile either:\n\t" + errfmt.Error())
			}
		}
	}
	return
}
