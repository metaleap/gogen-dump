package main

import (
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/loader"
	"os"
	"path/filepath"
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
			mf, lf := "me."+tf.FName, "l_"+tf.FName
			if ident, _ := fld.Type.(*ast.Ident); ident != nil {
				switch ident.Name {
				case "bool":
					tf.TmplW = "if " + mf + " { buf.WriteByte(1) } else { buf.WriteByte(0) }"
					tf.TmplR = mf + " = (data[i] == 1) ; i++"
				case "uint8", "byte":
					tf.TmplW = "buf.WriteByte(" + mf + ")"
					tf.TmplR = mf + " = data[i] ; i++"
				case "string":
					tf.TmplW = lf + " := uint64(len(" + mf + ")) ; " + writeLen(tf.FName) + " ; buf.WriteString(" + mf + ")"
				default:
					tf.TmplW = "//ident:" + ident.Name
					for tspec := range ts {
						if tspec.Name.Name == ident.Name {
							tdot.Types[i].HasWData = true
							tf.TmplW = mf + ".writeTo(&data) ; " + lf + " := uint64(data.Len()) ; " + writeLen(tf.FName) + " ; data.WriteTo(buf)"
							tf.TmplR = lf + " := int(*((*uint64)(unsafe.Pointer(&data[i])))) ; i += 8 ; " + mf + ".UnmarshalBinary(data[i : i+" + lf + "]) ; i += " + lf
							break
						}
					}
				}
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

func writeLen(fieldName string) string {
	return "buf.Write((*[8]byte)(unsafe.Pointer(&l_" + fieldName + "))[:])"
}
