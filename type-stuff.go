package main

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"unsafe"

	"github.com/go-forks/xxhash"
	"github.com/go-leap/str"
)

func collectTypes() {
	tdot.Structs = make([]*tmplDotStruct, 0, len(typeDefs))
	for t, s := range typeDefs {
		tds := &tmplDotStruct{TName: t.Name.Name, Fields: collectFields(s)}
		if l := len(tds.Fields); l > 0 {
			tds.Fields[l-1].isLast = true
			tdot.Structs = append(tdot.Structs, tds)
		}
	}

	// any tSynonyms we can pick up from struct-field-tags?
	for _, tds := range tdot.Structs {
		for _, tdf := range tds.Fields {
			if len(tdf.taggedUnion) == 1 {
				if tsyn, tref := finalElemTypeSpec(tdf.typeIdent), tdf.taggedUnion[0]; tsyn == tref {
					println(tds.TName + "." + tdf.FName + ": this type alias " + tdf.typeIdent + " -> " + tref + " was already known and could be removed")
				} else {
					typeSyns[tsyn] = tref
				}
				tdf.taggedUnion = nil
			} else if len(tdf.taggedUnion) > 1 {
				sort.Sort(tdf.taggedUnion)
			}
		}
	}

	tdot.allStructTypeDefsCollected = true
	sort.Sort(&tdot) // prevents pointless diffs, also for correct hashes

	// anylyze fixed-size fields for fixed-size siblings
	for _, tds := range tdot.Structs {
		fsstart, fsaccum := -1, 0
		for i, tdf := range tds.Fields {
			fs := tdf.fixedSize()
			if fs > 0 {
				if fsaccum += fs; fsstart < 0 {
					fsstart = i
				}
			}
			if numskip, notme := i-fsstart, fs <= 0; notme || tdf.isLast || tdf.nextOneWasSkipped {
				if notme {
					numskip--
				}
				if fsstart >= 0 && numskip > 0 {
					tds.Fields[fsstart].fixedsizeExt, tds.Fields[fsstart].fixedsizeExtNumSkip = fsaccum, numskip
				}
				fsstart, fsaccum = -1, 0
			}
		}
	}

	// prep some summary comments
	for _, tds := range tdot.Structs {
		add4hash := func(s string) { tds.hashInputSelf.writeString(s) }
		add4hash(s(len(tds.Fields)))
		for _, tdf := range tds.Fields {
			fs := tdf.fixedSize()
			add4hash(s(fs))
			tdf.Comment = tdf.finalTypeIdent()
			if len(tdf.taggedUnion) > 0 {
				tdf.Comment += " = [ " + ustr.Join(tdf.taggedUnion, " | ") + " ]"
			}
			add4hash(tdf.Comment)
			if fs > 0 {
				tdf.Comment += ", " + sizeStr(fs)
			}
			if tdf.fixedsizeExtNumSkip > 0 {
				tdf.Comment += ", begins fixed-size span of ~" + sizeStr(tdf.fixedsizeExt) + " (+padding/alignment..) over the next " + s(tdf.fixedsizeExtNumSkip) + " field(s)"
			}
			add4hash(s(tdf.fixedsizeExtNumSkip) + s(tdf.fixedsizeExt))
		}
		tds.Comment = s(len(tds.Fields)) + " field(s)"
		fs := tds.fixedSize()
		if fs > 0 {
			tds.Comment += ", always " + sizeStr(fs)
		}
		add4hash(s(fs))
	}

	// augment struct hashes with referenced-struct hashes
	forFieldsOfLocalStructs(func(fieldOwner *tmplDotStruct, fieldTypeRef *tmplDotStruct) {
		fieldTypeRef.hashInputSelf.copyTo(&fieldOwner.hashInputRefs)
	})
	forFieldsOfLocalStructs(func(fieldOwner *tmplDotStruct, fieldTypeRef *tmplDotStruct) {
		fieldTypeRef.hashInputRefs.copyTo(&fieldOwner.hashInputRefs)
	})
	for _, tds := range tdot.Structs {
		xxh := xxhash.New()
		tds.hashInputSelf.writeTo(xxh)
		tds.hashInputRefs.writeTo(xxh)
		tds.StructuralHash = xxh.Sum64()
	}
}

func forFieldsOfLocalStructs(on func(fieldOwner *tmplDotStruct, fieldTypeRef *tmplDotStruct)) {
	for _, tds := range tdot.Structs {
		for _, tdf := range tds.Fields {
			for _, tuspec := range tdf.taggedUnion {
				if tdstruc := tdot.byName(finalElemTypeSpec(tuspec)); tdstruc != nil {
					on(tds, tdstruc)
				}
			}
			if tdstruc := tdot.byName(finalElemTypeSpec(tdf.finalTypeIdent())); tdstruc != nil {
				on(tds, tdstruc)
			}
		}
	}
}

func sizeStr(size int) string {
	if size < 10000 {
		return s(size) + "b"
	}
	return s(size/1024) + "kb"
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
		var skip4inlinestruct bool
		if !tdf.skip {
			if substruc, _ := fld.Type.(*ast.StructType); substruc != nil {
				skip4inlinestruct, tdf.skip = true, true
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
		} else if lf := len(fields); lf > 0 && !skip4inlinestruct {
			fields[lf-1].nextOneWasSkipped = true
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
			pkg := goProg.Imported[pkgname]
			if pkg == nil {
				for pkgdep, pkginfo := range goProg.AllPackages {
					if pkgdep.Name() == pkgname {
						pkg = pkginfo
						break
					}
				}
			}
			tdot.Imps[pkgname].ImportPath = pkg.Pkg.Path()
		}
		return pkgname + "." + sel.Sel.Name, 0

	} else if iface, _ := t.(*ast.InterfaceType); iface != nil {
		return "interface{}", -1

	} else if fn, _ := t.(*ast.FuncType); fn != nil {
		return "", -1

	} else if struc, _ := t.(*ast.StructType); struc != nil {
		println("skipping a field: indirected (via ptr, slice, etc) inline in-struct anonymous sub-structs not supported (only directly placed ones are) — mark it `gogendump:\"-\"` to not show this message again.")
		return "", -1

	} else if ch, _ := t.(*ast.ChanType); ch != nil {
		return "", -1
	}
	panic(t)
}

func sizedArrMultAndElemType(arrTypeIdent string) (mult int, elemTypeIdent string) {
	mult, elemTypeIdent = 1, arrTypeIdent
	for elemTypeIdent[0] == '[' {
		if i := ustr.Pos(elemTypeIdent, "]"); i < 0 {
			return 1, ""
		} else if i == 1 {
			return
		} else if nulen, _ := strconv.Atoi(elemTypeIdent[1:i]); nulen <= 0 {
			return 1, ""
		} else if mult, elemTypeIdent = mult*nulen, elemTypeIdent[i+1:]; elemTypeIdent == "" {
			return 1, ""
		}
	}
	return
}

func fixedSizeForTypeSpec(typeIdent string) int {
	if ustr.IdxB(typeIdent, '*') >= 0 || ustr.Has(typeIdent, "[]") || ustr.Has(typeIdent, "map[") || typeIdent == "string" { // early return quite often
		return -1
	}
	mult, typeident := sizedArrMultAndElemType(typeIdent)
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
		if optVarintsNotFixedSize {
			return -1
		}
		return mult * int(unsafe.Sizeof(uint(0)))
	case "uintptr":
		if optVarintsNotFixedSize {
			return -1
		}
		return mult * int(unsafe.Sizeof(uintptr(0)))
	case "int":
		if optVarintsNotFixedSize {
			return -1
		}
		return mult * int(unsafe.Sizeof(int(0)))
	}
	if tsyn := typeSyns[typeident]; tsyn != "" {
		return mult * fixedSizeForTypeSpec(tsyn)
	} else if ustr.IdxB(typeident, '*') >= 0 || ustr.IdxB(typeident, '[') >= 0 || typeident == "string" {
		return -1
	}
	if tdot.allStructTypeDefsCollected {
		if tds := tdot.byName(typeident); tds != nil {
			return mult * tds.fixedSize()
		}
		return -1
	}
	return 0
}

func finalElemTypeSpec(typeSpec string) string {
	if typeSpec != "" {
		if typeSpec[0] == '*' {
			return finalElemTypeSpec(ustr.Skip(typeSpec, '*'))
		} else if pclose := ustr.IdxBMatching(typeSpec, ']', '['); pclose > 0 && (typeSpec[0] == '[' || ustr.Pref(typeSpec, "map[")) {
			return finalElemTypeSpec(typeSpec[pclose+1:])
		} else if tsyn := typeSyns[typeSpec]; tsyn != "" {
			return finalElemTypeSpec(tsyn)
		}
	}
	return typeSpec
}

func ensureImportFor(typeSpec string) (pkgName []string) {
	if typeSpec != "" {
		if typeSpec[0] == '*' {
			return ensureImportFor(ustr.Skip(typeSpec, '*'))
		} else if pclose := ustr.IdxBMatching(typeSpec, ']', '['); typeSpec[0] == '[' && pclose > 0 {
			return ensureImportFor(typeSpec[pclose+1:])
		} else if ustr.Pref(typeSpec, "map[") {
			return append(ensureImportFor(typeSpec[pclose+1:]),
				ensureImportFor(typeSpec[4:pclose])...)
		} else if i := ustr.IdxB(typeSpec, '.'); i > 0 {
			tdot.Imps[typeSpec[:i]].Used = true
			return []string{typeSpec[:i]}
		} else if tsyn := typeSyns[typeSpec]; tsyn != "" {
			return ensureImportFor(tsyn)
		}
	}
	return nil
}

func typeSizeHeur(typeIdent string, expr string) (heur *sizeHeuristics) {
	mult, tident := sizedArrMultAndElemType(typeIdent)
	if mult > 1 {
		expr = ""
	}
	if fs := fixedSizeForTypeSpec(tident); fs > 0 {
		heur = &sizeHeuristics{Lit: fs}
	} else if tident[0] == '*' {
		n := len(tident) - len(ustr.Skip(tident, '*'))
		heur = &sizeHeuristics{Op1: &sizeHeuristics{Lit: n}, OpAdd: true, Op2: typeSizeHeur(tident[n:], "")}
	} else if pclose := ustr.IdxBMatching(tident, ']', '['); tident[0] == '[' && pclose == 1 {
		exprlen := &sizeHeuristics{Lit: optHeuriticLenSlices}
		if expr != "" {
			exprlen = &sizeHeuristics{Expr: "len(" + expr + ")"}
		}
		heur = &sizeHeuristics{Op1: &sizeHeuristics{Lit: 8}, OpAdd: true, Op2: &sizeHeuristics{Op1: exprlen, OpMul: true, Op2: typeSizeHeur(tident[2:], "")}}
	} else if ustr.Pref(tident, "map[") {
		exprlen := &sizeHeuristics{Lit: optHeuristicLenMaps}
		if expr != "" {
			exprlen = &sizeHeuristics{Expr: "len(" + expr + ")"}
		}
		tkey, tval := tident[4:pclose], tident[pclose+1:]
		xkey := &sizeHeuristics{Op1: exprlen, OpMul: true, Op2: typeSizeHeur(tkey, "")}
		xval := &sizeHeuristics{Op1: exprlen, OpMul: true, Op2: typeSizeHeur(tval, "")}
		heur = &sizeHeuristics{Op1: &sizeHeuristics{Lit: 8}, OpAdd: true,
			Op2: &sizeHeuristics{Op1: xkey, OpAdd: true, Op2: xval}}
	} else if tident == "int" || tident == "uint" || tident == "uintptr" { // varints possibly not covered by above fixedSize handling
		heur = &sizeHeuristics{Lit: 8}
	} else if tident == "string" {
		exprlen := &sizeHeuristics{Lit: optHeuristicLenStrings}
		if expr != "" {
			exprlen = &sizeHeuristics{Expr: "len(" + expr + ")"}
		}
		heur = &sizeHeuristics{Op1: &sizeHeuristics{Lit: 8}, OpAdd: true, Op2: exprlen}
	} else if tsyn := typeSyns[tident]; tsyn != "" {
		heur = typeSizeHeur(tsyn, expr)
	} else {
		if expr != "" {
			expr += "."
		}
		if tds := tdot.byName(tident); tds != nil {
			heur = tds.sizeHeur(expr)
		}
	}

	if heur == nil {
		heur = &sizeHeuristics{Lit: optHeuristicSizeUnknowns}
	}
	if mult > 1 {
		heur = &sizeHeuristics{Op1: &sizeHeuristics{Lit: mult}, OpMul: true, Op2: heur}
	}
	return
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
