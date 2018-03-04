package main

import (
	"fmt"
	"go/ast"
	"go/token"
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
			if ident := typeIdent(fld.Type); ident != "" {
				tf.TmplR, tf.TmplW = genPrimRW(tf.FName, "", &tdot.Types[i], ident, 0, 0, taggedunion)
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

func typeIdent(t ast.Expr) string {
	if ident, _ := t.(*ast.Ident); ident != nil {
		return ident.Name
	} else if star, _ := t.(*ast.StarExpr); star != nil {
		return "*" + typeIdent(star.X)
	} else if arr, _ := t.(*ast.ArrayType); arr != nil {
		if lit, _ := arr.Len.(*ast.BasicLit); lit != nil && lit.Kind == token.INT {
			return "[" + lit.Value + "]" + typeIdent(arr.Elt)
		}
		return "[]" + typeIdent(arr.Elt)
	} else if sel, _ := t.(*ast.SelectorExpr); sel != nil {
		return sel.X.(*ast.Ident).Name + "." + sel.Sel.Name
	} else if iface, _ := t.(*ast.InterfaceType); iface != nil {
		return "interface{}"
	}
	panic(fmt.Sprintf("%T", t))
}

func genPrimRW(fieldName string, altNoMe string, t *tmplDotType, typeName string, numIndir int, iterDepth int, taggedUnion []string) (tmplR string, tmplW string) {
	nf, mfw, mfr, lf := fieldName, "me."+fieldName, "me."+fieldName, "l_"+fieldName
	if altNoMe != "" {
		if mfw = altNoMe; iterDepth > 0 {
			if mfr = ustr.TrimR(altNoMe, ":"); !ustr.Suff(mfr, "["+fieldName+"]") {
				mfr += "[" + fieldName + "]"
			}
		}
	}
	if numIndir > 0 {
		mfr = "v_" + nf + ":"
		mfw = "(" + ustr.Times("*", numIndir) + mfw + ")"
	}
	switch typeName {
	case "bool":
		tmplW = "if " + mfw + " { buf.WriteByte(1) } else { buf.WriteByte(0) }"
		tmplR = mfr + "= (data[pos] == 1) ; pos++"
	case "uint8", "byte":
		tmplW = "buf.WriteByte(" + mfw + ")"
		tmplR = mfr + "= data[pos] ; pos++"
	case "string":
		tmplW = lf + " := uint64(len(" + mfw + ")) ; " + genLenW(nf) + " ; buf.WriteString(" + mfw + ")"
		tmplR = genLenR(nf) + " ; " + mfr + "= string(data[pos : pos+" + lf + "]) ; pos += " + lf
	case "int16", "uint16":
		tmplW = genSizedW(nf, mfw, "2")
		tmplR = genSizedR(mfr, typeName, "2")
	case "rune", "int32", "float32", "uint32":
		tmplW = genSizedW(nf, mfw, "4")
		tmplR = genSizedR(mfr, typeName, "4")
	case "complex64", "float64", "uint64", "int64":
		tmplW = genSizedW(nf, mfw, "8")
		tmplR = genSizedR(mfr, typeName, "8")
	case "complex128":
		tmplW = genSizedW(nf, mfw, "16")
		tmplR = genSizedR(mfr, typeName, "16")
	case "uint", "uintptr":
		tmplW = genSizedW(nf, mfw, "uint64")
		tmplR = genSizedR(mfr, typeName, "uint64")
	case "int":
		tmplW = genSizedW(nf, mfw, "int64")
		tmplR = genSizedR(mfr, typeName, "int64")
	default:
		if typeName[0] == '*' {
			var numindir int
			for _, r := range typeName {
				if r == '*' {
					numindir++
				} else {
					break
				}
			}
			tr, tw := genPrimRW(nf, altNoMe, t, typeName[numindir:], numindir, iterDepth, taggedUnion)
			for i := 0; i < numindir; i++ {
				setnil := "; " + mfr + " = nil "
				if ustr.Pref(mfr, "v_") {
					setnil = ""
				}
				tmplR += "if data[pos] == 0 { pos++ " + setnil + "} else { pos++ ; "
				tmplW += "if " + ustr.Times("*", i) + mfw + " == nil { buf.WriteByte(0) } else { buf.WriteByte(1) ; "
			}
			tmplR += "\n\t\t" + tr + " ; "
			for i := 0; i < numindir; i++ {
				if i == 0 {
					if i == numindir-1 {
						tmplR += mfr + " = &v_" + nf
					} else {
						tmplR += "p0_" + nf + " := &v_" + nf + " ; "
					}
				} else if i == numindir-1 {
					tmplR += mfr + " = &p" + strconv.Itoa(i-1) + "_" + nf
				} else {
					tmplR += "p" + strconv.Itoa(i) + "_" + nf + " := &p" + strconv.Itoa(i-1) + "_" + nf + " ; "
				}
			}
			tmplR += "\n\t" + ustr.Times("}", numindir)
			tmplW += "\n\t\t" + tw
			tmplW += "\n\t" + ustr.Times("}", numindir)
		} else if pclose := ustr.Pos(typeName[1:], "]") + 1; typeName[0] == '[' && pclose > 0 {
			slen := typeName[1:pclose]
			if slen == "" {
				slen = "int(" + lf + ")"
				tmplR += genLenR(nf) + " ; " + mfr + "= make(" + typeName + ", " + lf + ") ; "
				tmplW += lf + " := uint64(len(" + mfw + ")) ; " + genLenW(nf) + " ; "
			} else if numIndir > 0 {
				tmplR += mfr + "= " + typeName + "{} ; "
			}
			idx := ustr.Times("i", iterDepth+1) + "_" + nf
			tmplR += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
			tmplW += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
			tr, _ := genPrimRW(idx, mfr, t, typeName[pclose+1:], 0, iterDepth+1, taggedUnion)
			tmplR += "\n\t\t" + tr
			_, tw := genPrimRW(idx, mfw+"["+idx+"]", t, typeName[pclose+1:], 0, iterDepth+1, taggedUnion)
			tmplW += "\n\t\t" + tw
			tmplR += "\n\t}"
			tmplW += "\n\t}"
		} else if len(taggedUnion) > 0 {
			tmplR += "t_" + nf + " := data[pos] ; pos++ ; switch t_" + nf + " {"
			tmplW += "switch t_" + nf + " := " + mfw + ".(type) {"
			for ti, tu := range taggedUnion {
				tr, _ := genPrimRW(nf, mfr, t, tu, 0, iterDepth, nil)
				tmplR += "\n\t\tcase " + strconv.Itoa(ti+1) + ":\n\t\t\t" + tr
				_, tw := genPrimRW(nf, "t_"+nf, t, tu, 0, iterDepth, nil)
				tmplW += "\n\t\tcase " + tu + ":\n\t\t\tbuf.WriteByte(" + strconv.Itoa(ti+1) + ") ; " + tw
			}
			tmplR += "\n\t\tdefault:\n\t\t\t" + mfr + " = nil"
			tmplW += "\n\t\tdefault:\n\t\t\tbuf.WriteByte(0)"
			tmplR += "\n\t}"
			tmplW += "\n\t}"
		} else {
			if ustr.Pref(mfr, "v_") {
				tmplR = mfr + "= " + typeName + "{} ; "
			}
			tmplR += genLenR(nf) + " ; if err = " + ustr.TrimR(mfr, ":") + ".UnmarshalBinary(data[pos : pos+l_" + nf + "]); err != nil { return } ; pos += l_" + nf
			tmplW += "d_" + nf + ", e_" + nf + " := " + mfw + ".MarshalBinary() ; if err = e_" + nf + "; err != nil { return } ; l_" + nf + " := uint64(len(d_" + nf + ")) ; " + genLenW(nf) + " ; buf.Write(d_" + nf + ")"

			for tspec := range ts {
				if tspec.Name.Name == typeName {
					t.HasWData = true
					if ustr.Pref(mfw, "(") && ustr.Suff(mfw, ")") && mfw[1] == '*' {
						mfw = ustr.TrimL(mfw[1:len(mfw)-1], "*")
					}
					tmplW = "if err = " + mfw + ".writeTo(&data); err != nil { return } ; " + lf + " := uint64(data.Len()) ; " + genLenW(nf) + " ; data.WriteTo(buf)"
					break
				}
			}
		}
	}
	return
}

func genSizedR(mfr string, typeName string, byteSize string) string {
	if byteSize == "int64" || byteSize == "uint64" {
		return mfr + "= " + typeName + "(*((*" + byteSize + ")(unsafe.Pointer(&data[pos])))) ; pos += 8"
	}
	return mfr + "= *((*" + typeName + ")(unsafe.Pointer(&data[pos]))) ; pos += " + byteSize
}

func genSizedW(fieldName string, mfw string, byteSize string) (s string) {
	if byteSize == "int64" || byteSize == "uint64" {
		return byteSize + "_" + fieldName + " := " + byteSize + "(" + mfw + ") ; buf.Write(((*[8]byte)(unsafe.Pointer(&" + byteSize + "_" + fieldName + ")))[:])"
	}
	return "buf.Write(((*[" + byteSize + "]byte)(unsafe.Pointer(&(" + mfw + "))))[:])"
}

func genLenR(fieldName string) string {
	return "l_" + fieldName + " := int(*((*uint64)(unsafe.Pointer(&data[pos])))) ; pos += 8"
}

func genLenW(fieldName string) string {
	return "buf.Write((*[8]byte)(unsafe.Pointer(&l_" + fieldName + "))[:])"
}
