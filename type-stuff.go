package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	// "reflect"
	"unsafe"

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
				if tdf.typeIdent, tdf.fixedSize = typeIdentAndFixedSize(fld.Type); tdf.typeIdent == "" {
					tdf.skip = true
				} else {
					tdf.isIfaceSlice = (tdf.typeIdent == "[]interface{}")
				}
			}
			if !tdf.skip {
				tdt.Fields = append(tdt.Fields, tdf)
			}
		}
		if l := len(tdt.Fields); l > 0 {
			tdt.Fields[l-1].isLast = true
			tdot.Types = append(tdot.Types, tdt)
		}
	}
}

// we go by type spec strings that we then later on 'parse' again, because they can also occur in struct-field-tags for tagged-unions
func typeIdentAndFixedSize(t ast.Expr) (typeSpec string, fixedSize int) {
	if ident, _ := t.(*ast.Ident); ident != nil {
		switch ident.Name {
		case "bool", "uint8", "byte":
			return ident.Name, 1
		case "int16", "uint16":
			return ident.Name, 2
		case "rune", "int32", "float32", "uint32":
			return ident.Name, 4
		case "complex64", "float64", "uint64", "int64":
			return ident.Name, 8
		case "complex128":
			return ident.Name, 16
		case "uint":
			if optVarintsInFixedSizeds {
				return ident.Name, int(unsafe.Sizeof(uint(0)))
			}
		case "uintptr":
			if optVarintsInFixedSizeds {
				return ident.Name, int(unsafe.Sizeof(uintptr(0)))
			}
		case "int":
			if optVarintsInFixedSizeds {
				return ident.Name, int(unsafe.Sizeof(int(0)))
			}
		}
		return ident.Name, 0

	} else if star, _ := t.(*ast.StarExpr); star != nil {
		if tident, _ := typeIdentAndFixedSize(star.X); tident != "" {
			return "*" + tident, 0
		}
		return "", 0

	} else if arr, _ := t.(*ast.ArrayType); arr != nil {
		if tident, fixedsize := typeIdentAndFixedSize(arr.Elt); tident != "" {
			if lit, _ := arr.Len.(*ast.BasicLit); lit != nil && lit.Kind == token.INT {
				arrlen, _ := strconv.Atoi(lit.Value)
				return "[" + lit.Value + "]" + tident, fixedsize * arrlen
			}
			return "[]" + tident, 0
		}
		return "", 0

	} else if ht, _ := t.(*ast.MapType); ht != nil {
		if tidkey, _ := typeIdentAndFixedSize(ht.Key); tidkey != "" {
			if tidval, _ := typeIdentAndFixedSize(ht.Value); tidval != "" {
				return "map[" + tidkey + "]" + tidval, 0
			}
		}
		return "", 0

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
		return pkgname + "." + sel.Sel.Name, 0

	} else if iface, _ := t.(*ast.InterfaceType); iface != nil {
		return "interface{}", 0

	} else if fn, _ := t.(*ast.FuncType); fn != nil {
		return "", 0

	} else if ch, _ := t.(*ast.ChanType); ch != nil {
		return "", 0
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
