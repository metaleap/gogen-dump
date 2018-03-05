package main

import (
	"fmt"
	"go/ast"
	"go/token"
	// "reflect"
	// "unsafe"

	"github.com/go-leap/dev/go"
	"github.com/go-leap/str"
)

func collectTypes() {
	tdot.Types = make([]*tmplDotType, 0, len(ts))
	for t, s := range ts {
		tdt := &tmplDotType{TName: t.Name.Name, Fields: make([]*tmplDotField, 0, len(s.Fields.List))}
		for _, fld := range s.Fields.List {
			tdf := &tmplDotField{}
			if l := len(fld.Names); l == 0 {
				if ident, _ := fld.Type.(*ast.Ident); ident != nil {
					tdf.FName = ident.Name
				} else {
					panic(fmt.Sprintf("%T", fld.Type))
				}
			} else if l == 1 {
				tdf.FName = fld.Names[0].Name
			} else {
				panic(l)
			}
			if fld.Tag != nil {
				if pos := ustr.Pos(fld.Tag.Value, "gogen-dump:\""); pos >= 0 {
					tagval := fld.Tag.Value[pos+12:]
					tagval = tagval[:ustr.Pos(tagval, "\"")]
					if tagval == "-" {
						tdf.skip = true
					} else {
						tdf.taggedUnion = ustr.Split(tagval, " ")
					}
				}
			}
			if !tdf.skip {
				if tdf.typeIdent, tdf.isFixedSize = typeIdent(fld.Type); tdf.typeIdent == "" {
					tdf.skip = true
				} else {
					tdf.isIfaceSlice = (tdf.typeIdent == "[]interface{}")
				}
			}
			if !tdf.skip {
				tdt.Fields = append(tdt.Fields, tdf)
			}
		}
		if len(tdt.Fields) > 0 {
			tdot.Types = append(tdot.Types, tdt)
		}
	}
}

// we go by type spec strings that we then later on 'parse' again, because they can also occur in struct-field-tags for tagged-unions
func typeIdent(t ast.Expr) (typeSpec string, isFixedSize bool) {
	if ident, _ := t.(*ast.Ident); ident != nil {
		switch ident.Name {
		case "bool", "uint8", "byte", "int16", "uint16", "rune", "int32", "float32", "uint32", "complex64", "float64", "uint64", "int64", "complex128", "uint", "uintptr", "int":
			return ident.Name, true
		}
		return ident.Name, false

	} else if star, _ := t.(*ast.StarExpr); star != nil {
		if tident, _ := typeIdent(star.X); tident != "" {
			return "*" + tident, false
		}
		return "", false

	} else if arr, _ := t.(*ast.ArrayType); arr != nil {
		if tident, fixedsize := typeIdent(arr.Elt); tident != "" {
			if lit, _ := arr.Len.(*ast.BasicLit); lit != nil && lit.Kind == token.INT {
				return "[" + lit.Value + "]" + tident, fixedsize
			}
			return "[]" + tident, false
		}
		return "", false

	} else if ht, _ := t.(*ast.MapType); ht != nil {
		if tidkey, _ := typeIdent(ht.Key); tidkey != "" {
			if tidval, _ := typeIdent(ht.Value); tidval != "" {
				return "map[" + tidkey + "]" + tidval, false
			}
		}
		return "", false

	} else if sel, _ := t.(*ast.SelectorExpr); sel != nil {
		pkgname := sel.X.(*ast.Ident).Name
		tdot.Imps[pkgname] = pkgname
		if udevgo.PkgsByImP == nil {
			if err := udevgo.RefreshPkgs(); err != nil {
				panic(err)
			}
		}
		if pkgimppath := ustr.Fewest(udevgo.PkgsByName(pkgname), "/", ustr.Shortest); pkgimppath != "" {
			tdot.Imps[pkgname] = pkgimppath
		}
		return pkgname + "." + sel.Sel.Name, false

	} else if iface, _ := t.(*ast.InterfaceType); iface != nil {
		return "interface{}", false

	} else if fn, _ := t.(*ast.FuncType); fn != nil {
		return "", false

	} else if ch, _ := t.(*ast.ChanType); ch != nil {
		return "", false
	}
	panic(fmt.Sprintf("%T", t))
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
