package main

import (
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/loader"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/go-leap/dev/go"
	"github.com/go-leap/str"
)

var (
	tdot        = tmplDotFile{ProgHint: "github.com/go-leap/gen/gogen-dump"}
	filePathSrc = udevgo.GopathSrc(tdot.ProgHint, "test-struct.go")
	typeNames   = []string{"testStruct", "embName"}
	ts          = map[*ast.TypeSpec]*ast.StructType{}
)

func main() {
	if len(os.Args) > 1 {
		if filePathSrc = os.Args[1]; len(os.Args) > 2 {
			typeNames = os.Args[2:]
		}
	}

	goast := loader.Config{}
	gofile, err := goast.ParseFile(filePathSrc, nil)
	tdot.PName = gofile.Name.Name
	if err != nil {
		panic(err)
	}
	for _, decl := range gofile.Decls {
		if gd, _ := decl.(*ast.GenDecl); gd != nil {
			for _, spec := range gd.Specs {
				if t, _ := spec.(*ast.TypeSpec); t != nil && t.Name != nil {
					if s, _ := t.Type.(*ast.StructType); s != nil {
						if len(typeNames) == 0 || ustr.In(t.Name.Name, typeNames...) {
							ts[t] = s
						}
					}
				}
			}
		}
	}
	if len(ts) == 0 {
		panic("none of the specified struct types could be found in Go file " + filePathSrc)
	} else {
		genDump()
	}
}

func genDump() {
	i, filePathDst := 0, filepath.Join(filepath.Dir(filePathSrc), ustr.TrimSuff(filepath.Base(filePathSrc), ".go")+".dump.go")

	tdot.Types = make([]tmplDotType, len(ts))
	var taggedunion []string
	for t, s := range ts {
		tdot.Types[i].TName = t.Name.Name
		tdot.Types[i].Fields = make([]tmplDotField, len(s.Fields.List))
		for f, fld := range s.Fields.List {
			tf := &tdot.Types[i].Fields[f]
			if l := len(fld.Names); l == 0 {
				if ident, _ := fld.Type.(*ast.Ident); ident != nil {
					tf.FName = ident.Name
				} else {
					panic(fmt.Sprintf("%T", fld.Type))
				}
			} else if l == 1 {
				tf.FName = fld.Names[0].Name
			} else {
				panic(l)
			}
			if taggedunion = nil; fld.Tag != nil {
				if pos := ustr.Pos(fld.Tag.Value, "gendump:\""); pos >= 0 {
					tagval := fld.Tag.Value[pos+9:]
					tagval = tagval[:ustr.Pos(tagval, "\"")]
					taggedunion = ustr.Split(tagval, " ")
				}
			}
			nf, mf := tf.FName, "me."+tf.FName
			if len(taggedunion) > 0 {
				tf.TmplR = "t_" + nf + " := data[i] ; i++ ; switch t_" + nf + " {"
				tf.TmplW = "switch t_" + nf + " := " + mf + ".(type) {"
				for ti, tu := range taggedunion {
					tr, tw := genPrimRW(tf.FName, "t_"+nf, &tdot.Types[i], tu)
					tf.TmplR += "\n\t\tcase " + strconv.Itoa(ti+1) + ":\n\t\t\t" + tr
					tf.TmplW += "\n\t\tcase " + tu + ":\n\t\t\tbuf.WriteByte(" + strconv.Itoa(ti+1) + ") ; " + tw
				}
				tf.TmplR += "\n\t\tdefault:\n\t\t\t" + mf + " = nil"
				tf.TmplW += "\n\t\tdefault:\n\t\t\tbuf.WriteByte(0)"
				tf.TmplR += "\n\t}"
				tf.TmplW += "\n\t}"
			} else if ident, _ := fld.Type.(*ast.Ident); ident != nil {
				tf.TmplR, tf.TmplW = genPrimRW(tf.FName, "", &tdot.Types[i], ident.Name)
			} else {
				tf.TmplW = "//no-ident:" + fmt.Sprintf("%T", fld.Type)
			}
		}
		i++
	}

	file, err := os.Create(filePathDst)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	tmpl := template.New(filePathDst)
	if _, err = tmpl.Parse(tmplPkg); err != nil {
		panic(err)
	} else if err = tmpl.Execute(file, &tdot); err != nil {
		panic(err)
	}
}

func genPrimRW(fieldName string, altNoMe string, t *tmplDotType, typeName string) (tmplR string, tmplW string) {
	nf, mf, mfr, lf := fieldName, "me."+fieldName, "me."+fieldName, "l_"+fieldName
	if altNoMe != "" {
		mf = altNoMe
	}
	switch typeName {
	case "bool":
		tmplW = "if " + mf + " { buf.WriteByte(1) } else { buf.WriteByte(0) }"
		tmplR = mfr + " = (data[i] == 1) ; i++"
	case "uint8", "byte":
		tmplW = "buf.WriteByte(" + mf + ")"
		tmplR = mfr + " = data[i] ; i++"
	case "string":
		tmplW = lf + " := uint64(len(" + mf + ")) ; " + genLenW(nf) + " ; buf.WriteString(" + mf + ")"
		tmplR = genLenR(nf) + " ; " + mfr + " = string(data[i : i+" + lf + "]) ; i += " + lf
	case "int16", "uint16":
		tmplW = genSizedW(nf, altNoMe, "2")
		tmplR = genSizedR(nf, typeName, "2")
	case "rune", "int32", "float32", "uint32":
		tmplW = genSizedW(nf, altNoMe, "4")
		tmplR = genSizedR(nf, typeName, "4")
	case "complex64", "float64", "uint64", "int64":
		tmplW = genSizedW(nf, altNoMe, "8")
		tmplR = genSizedR(nf, typeName, "8")
	case "complex128":
		tmplW = genSizedW(nf, altNoMe, "16")
		tmplR = genSizedR(nf, typeName, "16")
	case "uint", "uintptr":
		tmplW = genSizedW(nf, altNoMe, "uint64")
		tmplR = genSizedR(nf, typeName, "uint64")
	case "int":
		tmplW = genSizedW(nf, altNoMe, "int64")
		tmplR = genSizedR(nf, typeName, "int64")
	default:
		tmplW = "//ident:" + typeName
		for tspec := range ts {
			if tspec.Name.Name == typeName {
				t.HasWData = true
				tmplW = mf + ".writeTo(&data) ; " + lf + " := uint64(data.Len()) ; " + genLenW(nf) + " ; data.WriteTo(buf)"
				tmplR = genLenR(nf) + " ; " + mfr + ".UnmarshalBinary(data[i : i+" + lf + "]) ; i += " + lf
				break
			}
		}
	}
	return
}

func genSizedR(fieldName string, typeName string, byteSize string) string {
	if byteSize == "int64" || byteSize == "uint64" {
		return "me." + fieldName + " = " + typeName + "(*((*" + byteSize + ")(unsafe.Pointer(&data[i])))) ; i += 8"
	}
	return "me." + fieldName + " = *((*" + typeName + ")(unsafe.Pointer(&data[i]))) ; i += " + byteSize
}

func genSizedW(fieldName string, altNoMe string, byteSize string) (s string) {
	mf := "me." + fieldName
	if altNoMe != "" {
		mf = altNoMe
	}
	if byteSize == "int64" || byteSize == "uint64" {
		return byteSize + "_" + fieldName + " := " + byteSize + "(" + mf + ") ; buf.Write(((*[8]byte)(unsafe.Pointer(&" + byteSize + "_" + fieldName + ")))[:])"
	}
	return "buf.Write(((*[" + byteSize + "]byte)(unsafe.Pointer(&" + mf + ")))[:])"
}

func genLenR(fieldName string) string {
	return "l_" + fieldName + " := int(*((*uint64)(unsafe.Pointer(&data[i])))) ; i += 8"
}

func genLenW(fieldName string) string {
	return "buf.Write((*[8]byte)(unsafe.Pointer(&l_" + fieldName + "))[:])"
}
