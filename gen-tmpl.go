package main

import (
	"bytes"
	"go/format"
	"sort"
	"strconv"
	"text/template"
)

const tmplSrc = `package {{.PName}}
// Code generated by {{.ProgHint}} - DO NOT EDIT.
{{ $bBytes := .BBuf.Bytes }}
{{ $bWriteTo:= .BBuf.WriteTo }}
{{ $bLen := .BBuf.Len }}
{{ $bCtor := .BBuf.Ctor }}
{{ $bType := .BBuf.Type }}

// This file consists {{if .BBuf.Stdlib}}solely{{else}}largely{{end}} of generated (de)serialization methods for the following {{len .Structs}} struct type(s).
{{range .Structs}}// - {{.TName}} (signature: {{.StructuralHash}})
{{end}}
import (
	{{if .BBuf.Stdlib}}"bytes"{{end}}
	"errors"
	"io"
	"unsafe"
	{{range $pkgname, $pkg := .Imps}}{{if $pkg.Used}}
		{{ $pkgname }} "{{$pkg.ImportPath}}"{{end}}{{end}}
)

{{range .Structs}}

/* {{.TName}}:
   {{.Comment}}

   The serialization view:
{{range .Fields}}   - {{.FName}} - {{.Comment}}
{{end}}*/

func (me *{{.TName}}) marshalTo(buf {{$bType}}) (err error) {
	{{if .TmplW}}
	{{.TmplW}}
	{{else}}
	{{range .Fields}}
	{{.TmplW}}
	{{end}}
	{{end}}
	return
}

// MarshalBinary` + " implements `encoding.BinaryMarshaler` by serializing `me` into `data` (that can be consumed by `UnmarshalBinary`)" + `.
func (me *{{.TName}}) MarshalBinary() (data []byte, err error) {
	buf := {{$bCtor}}(make([]byte, 0, {{.InitialBufSize}}))
	if err = me.marshalTo(buf); err == nil {
		data = {{$bBytes}}
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

// UnmarshalBinary` + " implements `encoding.BinaryUnmarshaler` by deserializing from `data` (that was originally produced by `MarshalBinary`) into `me`" + `.
func (me *{{.TName}}) UnmarshalBinary(data []byte) (err error) {
	var pos0 int
	err = me.unmarshalFrom(&pos0, data)
	return
}

// ReadFrom` + " implements `io.ReaderFrom` by deserializing from `r` into `me`" + `.
// It reads only as many bytes as indicated necessary in the initial 16-byte header prefix from ` + "`WriteTo`" + `, any remainder remains unread.
func (me *{{.TName}}) ReadFrom(r io.Reader) (int64, error) {
	var header [2]uint64
	n, err := io.ReadAtLeast(r, ((*[16]byte)(unsafe.Pointer(&header[0])))[:], 16)
	if err == nil {
		if header[0] != {{.StructuralHash}} {
			err = errors.New("{{.TName}}: incompatible signature header")
		} else {
			data := make([]byte, header[1])
			if n, err = io.ReadAtLeast(r, data, len(data)); err == nil {
				var pos0 int
				err = me.unmarshalFrom(&pos0, data)
			}
			n += 16
		}
	}
	return int64(n), err
}

// WriteTo` + " implements `io.WriterTo` by serializing `me` to `w`" + `.
// ` + "`WriteTo` and `ReadFrom` rely on a 16-byte header prefix to the subsequent raw serialization data handled by `MarshalBinary`/`UnmarshalBinary` respectively." + `
func (me *{{.TName}}) WriteTo(w io.Writer) (n int64, err error) {
	buf := {{$bCtor}}(make([]byte, 0, {{.InitialBufSize}}))
	if err = me.marshalTo(buf); err == nil {
		header := [2]uint64 { {{.StructuralHash}}, uint64({{$bLen}}) }
		var l int
		if l, err = w.Write(((*[16]byte)(unsafe.Pointer(&header[0])))[:]); err != nil {
			n = int64(l)
		} else {
			n, err = {{$bWriteTo}}(w)
			n += 16
		}
	}
	return
}
{{end}}

{{if not .BBuf.Stdlib}}

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
		me.b = make([]byte, l+1, l+l+128) // the constant extra padding: if we're tiny (~0), it helps much; if we're large (MBs), it hurts none
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
{{end}}
`

type tmplDotFile struct {
	ProgHint string
	PName    string
	Structs  []*tmplDotStruct
	Imps     map[string]*tmplDotPkgImp
	BBuf     struct {
		Stdlib  bool
		Bytes   string
		Ctor    string
		Len     string
		Type    string
		WriteB  string
		WriteS  string
		WriteN  string
		WriteTo string
	}

	allStructTypeDefsCollected bool
}

func (me *tmplDotFile) Len() int               { return len(me.Structs) }
func (me *tmplDotFile) Less(i int, j int) bool { return me.Structs[i].TName < me.Structs[j].TName }
func (me *tmplDotFile) Swap(i int, j int)      { me.Structs[i], me.Structs[j] = me.Structs[j], me.Structs[i] }

func (me *tmplDotFile) byName(name string) *tmplDotStruct {
	for _, tds := range me.Structs {
		if tds.TName == name {
			return tds
		}
	}
	return nil
}

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
	StructuralHash uint64
	Comment        string

	hashInputSelf writeBuf
	hashInputRefs writeBuf

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

func (me *tmplDotStruct) sizeHeur(exprPref string) *sizeHeuristics {
	if me.sizeheuring {
		return &sizeHeuristics{Lit: optHeuristicSizeUnknowns}
	}
	me.sizeheuring = true
	if fs := me.fixedSize(); fs > 0 {
		return &sizeHeuristics{Lit: fs}
	}
	var last *sizeHeuristics
	for _, tdf := range me.Fields {
		this := tdf.sizeHeur(exprPref)
		if last == nil {
			last = this
		} else {
			last = &sizeHeuristics{Op1: last, OpAdd: true, Op2: this}
		}
	}
	me.sizeheuring = false
	return last
}

type tmplDotField struct {
	FName   string
	TmplW   string
	TmplR   string
	Comment string

	typeIdent         string
	taggedUnion       sort.StringSlice
	skip              bool
	nextOneWasSkipped bool
	isLast            bool

	fixedsize           int
	fixedsizeExt        int
	fixedsizeExtNumSkip int
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

func (me *tmplDotField) sizeHeur(exprPref string) *sizeHeuristics {
	if fs := me.fixedSize(); fs > 0 {
		return &sizeHeuristics{Lit: fs}
	} else {
		if exprPref != "" {
			exprPref += me.FName
		}
		return typeSizeHeur(me.finalTypeIdent(), exprPref)
	}
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
				println("be warned! the generated code could not be formatted, so it won't compile either:\n\t" + errfmt.Error())
			}
		}
	}
	return
}

type sizeHeuristics struct {
	Expr  string
	Lit   int
	OpMul bool
	OpAdd bool
	Op1   *sizeHeuristics
	Op2   *sizeHeuristics
}

func (me *sizeHeuristics) isLit() bool {
	return (!me.OpAdd) && (!me.OpMul) && me.Expr == ""
}

func (me *sizeHeuristics) reduce() *sizeHeuristics {
	if me.Op1 != nil && me.Op2 != nil {
		me.Op1, me.Op2 = me.Op1.reduce(), me.Op2.reduce()
		l1, l2 := me.Op1.Lit, me.Op2.Lit
		o1, o2 := me.Op1.isLit(), me.Op2.isLit()
		switch {
		case me.OpMul:

			if l1 == 1 {
				return me.Op2
			} else if l2 == 1 {
				return me.Op1
			} else if (o1 && l1 == 0) || (o2 && l2 == 0) {
				return &sizeHeuristics{}
			} else if o1 && o2 {
				return &sizeHeuristics{Lit: l1 * l2}
			}
		case me.OpAdd:
			if o1 && l1 == 0 {
				return me.Op2
			} else if o2 && l2 == 0 {
				return me.Op1
			} else if o1 && o2 {
				return &sizeHeuristics{Lit: l1 + l2}
			}
		}
	}
	return me
}

func (me *sizeHeuristics) String() string {
	switch {
	case me.isLit():
		return strconv.Itoa(me.Lit)
	case me.Expr != "":
		return me.Expr
	case me.OpMul:
		return "(" + me.Op1.String() + " * " + me.Op2.String() + ")"
	case me.OpAdd:
		return "(" + me.Op1.String() + " + " + me.Op2.String() + ")"
	}
	panic("forgot a case in switch?!")
}
