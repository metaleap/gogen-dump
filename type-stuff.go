package main

import (
	"go/ast"
	"go/token"
	"strconv"
	// "reflect"
	"unsafe"

	"github.com/go-leap/dev/go"
	"github.com/go-leap/str"
)

func collectTypes() {
	tdot.Structs = make([]*tmplDotStructTypeDef, 0, len(ts))
	for t, s := range ts {
		tdstd := &tmplDotStructTypeDef{TName: t.Name.Name, Fields: collectFields(s)}
		if l := len(tdstd.Fields); l > 0 {
			tdstd.Fields[l-1].isLast = true
			tdot.Structs = append(tdot.Structs, tdstd)
		}
	}
	for _, tdstd := range tdot.Structs {
		for _, tdf := range tdstd.Fields {
			if len(tdf.taggedUnion) == 1 {
				if tsyn, tref := finalElemTypeSpec(tdf.typeIdent), tdf.taggedUnion[0]; tsyn == tref {
					panic(tdstd.TName + "." + tdf.FName + ": remove the redundant type alias of " + tdf.typeIdent + " = " + tref)
				} else {
					tSynonyms[tsyn] = tref
				}
				tdf.taggedUnion = nil
			}
		}
	}
}

func collectFields(st *ast.StructType) (fields []*tmplDotField) {
	fields = make([]*tmplDotField, 0, len(st.Fields.List))
	for _, fld := range st.Fields.List {
		tdf := &tmplDotField{}
		if l := len(fld.Names); l == 0 {
			if ident, _ := fld.Type.(*ast.Ident); ident != nil {
				tdf.FName = ident.Name
			} else {
				panic(fld.Type)
			}
		} else if l == 1 {
			tdf.FName = fld.Names[0].Name
		} else {
			panic(l)
		}
		if tagpref := "ggd:\""; fld.Tag != nil {
			if pos := ustr.Pos(fld.Tag.Value, tagpref); pos >= 0 {
				tagval := fld.Tag.Value[pos+len(tagpref):]
				if tagval = ustr.Trim(tagval[:ustr.Pos(tagval, "\"")]); tagval == "-" {
					tdf.skip = true
				} else if tdf.taggedUnion = ustr.Sans(ustr.Map(ustr.Split(ustr.Trim(tagval), " "), ustr.Trim), " ", ""); len(tdf.taggedUnion) > 255 {
					panic(tdf.FName + ": too many case alternatives for serializable .(type) switch (maximum is 255)")
				}
			}
		}
		if !tdf.skip {
			if substruc, _ := fld.Type.(*ast.StructType); substruc != nil {
				tdf.skip = true
				for _, subtdf := range collectFields(substruc) {
					subtdf.FName = tdf.FName + "." + subtdf.FName
					fields = append(fields, subtdf)
				}
			} else if tdf.typeIdent, tdf.fixedsize = typeIdentAndFixedSize(fld.Type); tdf.typeIdent == "" {
				tdf.skip = true
			}
		}
		if !tdf.skip {
			fields = append(fields, tdf)
		}
	}
	return
}

// we go by type spec strings that we then later on 'parse' again, because they can also occur in struct-field-tags for tagged-unions
func typeIdentAndFixedSize(t ast.Expr) (typeSpec string, fixedSize int) {
	if ident, _ := t.(*ast.Ident); ident != nil {
		return ident.Name, fixedSizeForTypeSpec(ident.Name)

	} else if star, _ := t.(*ast.StarExpr); star != nil {
		if tident, _ := typeIdentAndFixedSize(star.X); tident != "" {
			return "*" + tident, -1
		}
		return "", -1

	} else if arr, _ := t.(*ast.ArrayType); arr != nil {
		if tident, fixedsize := typeIdentAndFixedSize(arr.Elt); tident != "" {
			if lit, _ := arr.Len.(*ast.BasicLit); lit != nil && lit.Kind == token.INT {
				if arrlen, _ := strconv.Atoi(lit.Value); arrlen >= 0 {
					fixedsize *= arrlen
				} else {
					return "", -1
				}
				return "[" + lit.Value + "]" + tident, fixedsize
			}
			return "[]" + tident, -1
		}
		return "", -1

	} else if ht, _ := t.(*ast.MapType); ht != nil {
		if tidkey, _ := typeIdentAndFixedSize(ht.Key); tidkey != "" {
			if tidval, _ := typeIdentAndFixedSize(ht.Value); tidval != "" {
				return "map[" + tidkey + "]" + tidval, -1
			}
		}
		return "", -1

	} else if sel, _ := t.(*ast.SelectorExpr); sel != nil {
		pkgname := sel.X.(*ast.Ident).Name
		if tdot.Imps[pkgname] == nil {
			tdot.Imps[pkgname] = &tmplDotPkgImp{ImportPath: pkgname}
			if udevgo.PkgsByImP == nil {
				if err := udevgo.RefreshPkgs(); err != nil {
					panic(err)
				}
			}
			if pkgimppath := ustr.Fewest(udevgo.PkgsByName(pkgname), "/", ustr.Shortest); pkgimppath != "" {
				tdot.Imps[pkgname].ImportPath = pkgimppath
			}
		}
		return pkgname + "." + sel.Sel.Name, -1

	} else if iface, _ := t.(*ast.InterfaceType); iface != nil {
		return "interface{}", -1

	} else if fn, _ := t.(*ast.FuncType); fn != nil {
		return "", -1

	} else if struc, _ := t.(*ast.StructType); struc != nil {
		println("skipping a field: indirect (via ptr, slice, etc) in-struct inline sub-structs not supported (only direct ones are). mark it `gogendump:\"-\"` to not see this message.")
		return "", -1

	} else if ch, _ := t.(*ast.ChanType); ch != nil {
		return "", -1
	}
	panic(t)
}

func fixedSizeArrMult(arrTypeIdent string) (mult int, elemTypeIdent string) {
	mult, elemTypeIdent = 1, arrTypeIdent
	for elemTypeIdent[0] == '[' {
		if i := ustr.Pos(elemTypeIdent, "]"); i < 0 {
			return 1, ""
		} else if i == 1 {
			return 1, elemTypeIdent[2:]
		} else if nulen, _ := strconv.Atoi(elemTypeIdent[1:i]); nulen <= 0 {
			return 1, ""
		} else if mult, elemTypeIdent = mult*nulen, elemTypeIdent[i+1:]; elemTypeIdent == "" {
			return 1, ""
		}
	}
	return
}

func fixedSizeForTypeSpec(typeIdent string) int {
	mult, typeident := fixedSizeArrMult(typeIdent)
	switch typeident {
	case "bool", "uint8", "byte", "int8":
		return mult * 1
	case "int16", "uint16":
		return mult * 2
	case "rune", "int32", "float32", "uint32":
		return mult * 4
	case "complex64", "float64", "uint64", "int64":
		return mult * 8
	case "complex128":
		return mult * 16
	case "uint":
		if optVarintsInFixedSizeds {
			return mult * int(unsafe.Sizeof(uint(0)))
		}
		return -1
	case "uintptr":
		if optVarintsInFixedSizeds {
			return mult * int(unsafe.Sizeof(uintptr(0)))
		}
		return -1
	case "int":
		if optVarintsInFixedSizeds {
			return mult * int(unsafe.Sizeof(int(0)))
		}
		return -1
	}
	if tsyn := tSynonyms[typeident]; tsyn != "" {
		return mult * fixedSizeForTypeSpec(tsyn)
	} else if ustr.Idx(typeident, '*') >= 0 || ustr.Idx(typeident, '[') >= 0 {
		return -1
	}
	if tdot.allStructTypeDefsCollected {
		for _, tdstd := range tdot.Structs {
			if tdstd.TName == typeident {
				return mult * tdstd.fixedSize()
			}
		}
		return -1
	}
	return 0
}

func finalElemTypeSpec(typeSpec string) string {
	if typeSpec != "" {
		if typeSpec[0] == '*' {
			return finalElemTypeSpec(ustr.TrimL(typeSpec, "*"))
		} else if pclose := ustr.Idx(typeSpec, ']'); pclose > 0 && (typeSpec[0] == '[' || ustr.Pref(typeSpec, "map[")) {
			return finalElemTypeSpec(typeSpec[pclose+1:])
		} else if tsyn := tSynonyms[typeSpec]; tsyn != "" {
			return finalElemTypeSpec(tsyn)
		}
	}
	return typeSpec
}

func ensureImportFor(typeSpec string) (pkgName []string) {
	if typeSpec != "" {
		if typeSpec[0] == '*' {
			return ensureImportFor(ustr.TrimL(typeSpec, "*"))
		} else if pclose := ustr.Idx(typeSpec, ']'); typeSpec[0] == '[' && pclose > 0 {
			return ensureImportFor(typeSpec[pclose+1:])
		} else if ustr.Pref(typeSpec, "map[") {
			return append(ensureImportFor(typeSpec[pclose+1:]),
				ensureImportFor(typeSpec[4:pclose])...)
		} else if i := ustr.Idx(typeSpec, '.'); i > 0 {
			tdot.Imps[typeSpec[:i]].Used = true
			return []string{typeSpec[:i]}
		} else if tsyn := tSynonyms[typeSpec]; tsyn != "" {
			return ensureImportFor(tsyn)
		}
	}
	return nil
}

// // nicked from teh_cmc/gools/zerocopy:
// // converts a string to a []byte without any copy.
// // NOTE: do not ever modify the returned byte slice.
// // NOTE: do not ever use the returned byte slice once the original string went
// // out of scope.
// func zeroCopyStringToBytes(s string) (b []byte) {
// 	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
// 	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
// 	bh.Len, bh.Cap, bh.Data = sh.Len, sh.Len, sh.Data
// 	return
// }

// // nicked from teh_cmc/gools/zerocopy:
// // converts a []byte to a string without any copy.
// // NOTE: do not ever use the returned string once the original []byte went
// // out of scope.
// func zeroCopyBytesToString(b []byte) (s string) {
// 	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
// 	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
// 	sh.Len, sh.Data = bh.Len, bh.Data
// 	return
// }
