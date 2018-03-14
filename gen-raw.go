package main

import (
	"path/filepath"
	"strconv"

	"github.com/go-leap/fs"
	"github.com/go-leap/str"
)

var (
	s = strconv.Itoa
)

func genDump() error {
	repl := ustr.Repl(
		"b·writeN", tdot.BBuf.WriteN,
		"b·writeB", tdot.BBuf.WriteB,
		"b·writeS", tdot.BBuf.WriteS,
	)
	for _, tdt := range tdot.Structs {
		if fs := tdt.fixedSize(); fs > 0 && !optNoFixedSizeCode {
			ensureImportFor(tdt.TName)
			tdt.TmplR = "*me = *((*" + tdt.TName + ")(unsafe.Pointer(&data[p]))) ; p += " + s(fs)
			tdt.TmplW = "b·writeN((*[" + s(fs) + "]byte)(unsafe.Pointer(me))[:])"
		} else {
			for i := 0; i < len(tdt.Fields); i++ {
				tdf := tdt.Fields[i]
				oneop := tdf.fixedsizeExtNumSkip > 0
				if oneop && !optNoFixedSizeCode {
					addr, fse := "&me."+tdf.FName, s(tdf.fixedsizeExt)
					if i += tdf.fixedsizeExtNumSkip; tdf.typeIdent[0] == '[' {
						addr += "[0]"
					}
					tdf.TmplW = "b·writeN( (*[" + fse + "]byte)(unsafe.Pointer(" + addr + ")) [:] )"
					tdf.TmplR = "*( (*[" + fse + "]byte)(unsafe.Pointer(" + addr + ")) ) = *( (*[" + fse + "]byte)(unsafe.Pointer(&data[p])) ) ; p += " + fse + " "
				} else {
					tdf.TmplR, tdf.TmplW = genForFieldOrVarOfNamedTypeRW(tdf.FName, "", tdt, tdf.typeIdent, "", 0, 0, tdf.taggedUnion)
				}
				tdf.TmplR, tdf.TmplW = repl.Replace(tdf.TmplR), repl.Replace(tdf.TmplW)
			}
		}
		tdt.InitialBufSize = tdt.sizeHeur("me.").reduce().String()
		tdt.TmplR, tdt.TmplW = repl.Replace(tdt.TmplR), repl.Replace(tdt.TmplW)
	}

	genFileName = filepath.Join(goPkgDirPath, genFileName)
	src, err := genViaTmpl()
	if err == nil {
		err = ufs.WriteBinaryFile(genFileName, src)
	}
	return err
}

func genForFieldOrVarOfNamedTypeRW(fieldName string, altNoMe string, tds *tmplDotStruct, typeName string, tmpVarPref string, numIndir int, iterDepth int, taggedUnion []string) (tmplR string, tmplW string) {
	nf, mfw, mfr, lf, nfr := fieldName, "me."+fieldName, "me."+fieldName, "l"+ustr.Replace(fieldName, ".", ""), ustr.Replace(fieldName, ".", "")
	if altNoMe != "" {
		if mfw = altNoMe; iterDepth > 0 {
			if mfr = ustr.Drop(altNoMe, ':'); !ustr.Suff(mfr, "["+nf+"]") {
				mfr += "[" + nf + "]"
			}
		}
	}
	if numIndir > 0 {
		if mfr = "v" + s(numIndir) + s(iterDepth) + s(len(taggedUnion)) + ":"; tmpVarPref != "" {
			mfw = tmpVarPref + s(numIndir-1) + s(iterDepth) + s(len(taggedUnion))
		}
	} else if altNoMe == "" && tmpVarPref != "" {
		mfr = tmpVarPref // + nfr
	} else if numIndir == 0 && iterDepth > 0 && altNoMe == "" && (ustr.Pref(nf, "k") || ustr.Pref(nf, "m")) {
		mfr = "b" + nf
	}
	mfwd := mfw
	if numIndir > 0 {
		mfwd = "(" + ustr.Times("*", numIndir) + mfw + ")"
	}
	var cast string
	if optSafeVarints {
		cast = "uint64"
	}
	switch typeName {
	case "bool":
		tmplW = "if " + mfwd + " { b·writeB(1) } else { b·writeB(0) }"
		tmplR = mfr + "= (data[p] == 1) ; p++ "
	case "uint8", "byte":
		tmplW = "b·writeB(" + mfwd + ")"
		tmplR = mfr + "= data[p] ; p++ "
	case "int8":
		tmplW = genSizedW(nfr, mfwd, "1")
		tmplR = genSizedR(mfr, typeName, "1")
	case "string":
		tmplW = lf + " := " + cast + "(len(" + mfw + ")) ; " + genLenW(nfr) + " ; b·writeS(" + mfw + ")"
		tmplR = genLenR(nfr) + " ; " + mfr + "= string(data[p : p+" + lf + "]) ; p += " + lf
		if iterDepth == 0 {
			if tmplW = "{ " + tmplW + " }"; !ustr.Pref(mfr, "v") {
				tmplR = "{ " + tmplR + " }"
			}
		}
	case "int16", "uint16":
		tmplW = genSizedW(nfr, mfwd, "2")
		tmplR = genSizedR(mfr, typeName, "2")
	case "rune", "int32", "float32", "uint32":
		tmplW = genSizedW(nfr, mfwd, "4")
		tmplR = genSizedR(mfr, typeName, "4")
	case "complex64", "float64", "uint64", "int64":
		tmplW = genSizedW(nfr, mfwd, "8")
		tmplR = genSizedR(mfr, typeName, "8")
	case "complex128":
		tmplW = genSizedW(nfr, mfwd, "16")
		tmplR = genSizedR(mfr, typeName, "16")
	case "uint", "uintptr":
		tmplW = genSizedW(nfr, mfwd, "uint64")
		tmplR = genSizedR(mfr, typeName, "uint64")
	case "int":
		tmplW = genSizedW(nfr, mfwd, "int64")
		tmplR = genSizedR(mfr, typeName, "int64")
	default:
		if typeName[0] == '*' {
			// POINTER

			numindir := len(typeName) - len(ustr.Skip(typeName, '*'))
			tn, id := typeName[numindir:], s(iterDepth)+s(len(taggedUnion)) //+s(numindir)+s(numIndir)
			vpref, pv0 := "pv", ustr.Pref(tn, "[]") || ustr.Pref(tn, "map[") || tn == "string"
			if !pv0 {
				vpref = ""
			}
			tr, tw := genForFieldOrVarOfNamedTypeRW(nf, altNoMe, tds, tn, vpref, numindir, iterDepth, taggedUnion)
			tmplR = "{ "
			for i := 0; i < numindir; i++ {
				tmplR += " var p" + s(i) + id + " " + ustr.Times("*", numindir-i) + tn + " ; "
			}
			for i := 0; i < numindir; i++ {
				tmplR += "if p++; data[p-1] != 0 { "
				if i == 0 {
					if tmplW += "if " + mfw + " == nil { b·writeB(0) } else { b·writeB(1) ; "; pv0 {
						tmplW += "pv0" + id + " := *" + mfw + " ; " // } else {tmplW += " /*pv0*/ "
					}
				} else {
					tmplW += "if pv" + s(i-1) + id + " == nil { b·writeB(0) } else { b·writeB(1) ; pv" + s(i) + id + " := *pv" + s(i-1) + id + " ; "
				}
			}
			tmplR += "\n\t\t" + tr + " ; "
			for i := numindir - 1; i >= 0; i-- {
				if i == numindir-1 {
					tmplR += " ; p" + s(i) + id + " = &v" + s(numindir) + id + " ; "
				} else {
					tmplR += " ; p" + s(i) + id + " = &p" + s(i+1) + id + " ; "
				}
				tmplR += "\n\t}\n\t"
			}
			tmplR += " ; " + mfr + " = p0" + id + " } "
			tmplW += "\n\t\t" + tw
			tmplW += "\n\t" + ustr.Times("}", numindir)
		} else if ismap, pclose := ustr.Pref(typeName, "map["), ustr.IdxBMatching(typeName, ']', '['); pclose > 0 && (typeName[0] == '[' || ismap) {
			// ARRAY / SLICE / MAP

			arrfixedsize, slen, hasl := 0, typeName[1:pclose], false
			if slen == "" || ismap {
				if hasl, slen = true, "("+lf+")"; optSafeVarints {
					slen = "int" + slen
				}
				ensureImportFor(typeName)
				tmplR += genLenR(nfr) + " ; " + mfr + "= make(" + typeName + ", " + lf + ") ; "
				tmplW += lf + " := " + cast + "(len(" + mfw + ")) ; " + genLenW(nfr) + " ; "
			} else {
				if numIndir > 0 {
					ensureImportFor(typeName)
					tmplR += mfr + "= " + typeName + "{} ; "
				}
				arrfixedsize = fixedSizeForTypeSpec(typeName)
			}
			valtypespec, idx := typeName[pclose+1:], "i"+s(iterDepth)

			if ismap {
				keytypespec, mk, mv := typeName[4:pclose], "k"+s(iterDepth), "m"+s(iterDepth)
				tmplR += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
				tmplW += "for " + mk + ", " + mv + " := range " + mfw + " {"

				tmplR += "\n\t\tvar b" + mk + " " + keytypespec
				tmplR += "\n\t\tvar b" + mv + " " + valtypespec
				tr, _ := genForFieldOrVarOfNamedTypeRW(mk, "", tds, keytypespec, "", 0, iterDepth+1, nil)
				tmplR += "\n\t\t" + tr
				tr, _ = genForFieldOrVarOfNamedTypeRW(mv, "", tds, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplR += "\n\t\t" + tr
				tmplR += "\n\t\t" + ustr.Drop(mfr, ':') + "[b" + mk + "] = b" + mv
				tmplR += "\n\t}"

				_, tw := genForFieldOrVarOfNamedTypeRW(mk, mk, tds, keytypespec, "", 0, iterDepth+1, nil)
				tmplW += "\n\t\t" + tw
				_, tw = genForFieldOrVarOfNamedTypeRW(mv, mv, tds, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplW += "\n\t\t" + tw
				tmplW += "\n\t}"
			} else if afs := s(arrfixedsize); arrfixedsize > 0 && !optNoFixedSizeCode {
				tmplW = genSizedW(nfr, mfwd+"[0]", afs)
				tmplR = genSizedR(mfr, typeName, afs)
				return
			} else {
				offr, offw := len(tmplR), len(tmplW)
				if valtypespec == "byte" || valtypespec == "uint8" {
					tmplR += "copy(" + ustr.Drop(mfr, ':') + "[:" + slen + "], data[p:" + slen + "]) ; p += " + slen
				} else {
					tmplR += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
					tr, _ := genForFieldOrVarOfNamedTypeRW(idx, mfr, tds, valtypespec, "", 0, iterDepth+1, taggedUnion)
					tmplR += "\n\t\t" + tr + "\n\t}"
				}
				tmplW += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
				_, tw := genForFieldOrVarOfNamedTypeRW(idx, mfw+"["+idx+"]", tds, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplW += "\n\t\t" + tw + "\n\t}"

				if fs := fixedSizeForTypeSpec(valtypespec); fs > 0 && optFixedSizeMaxSizeInGB > 0 && !optNoFixedSizeCode {
					sfs, fixedsizemax := s(fs), int(optFixedSizeMaxSizeInGB*(1024*1024*1024))
					maxlen := (fixedsizemax / fs) + 1
					tmplW = tmplW[:offw] + " ; if " + slen + " > 0 && " + slen + " < " + s(maxlen) + " { " +
						" b·writeN((*[" + s(fixedsizemax-1) + "]byte)(unsafe.Pointer(&" + ustr.Drop(mfw, ':') + "[0]))[:" + sfs + "*" + slen + "]) " +
						" } else { " + tmplW[offw:] + "} ; "
					tmplR = tmplR[:offr] + " ; if " + slen + " > 0 && " + slen + " < " + s(maxlen) + " { " +
						" lmul := " + sfs + "*" + slen + " ; copy(((*[" + s(fixedsizemax-1) + "]byte)(unsafe.Pointer(&" + ustr.Drop(mfr, ':') + "[0])))[0:lmul], data[p:p+(lmul)])  ; p += (lmul) " +
						" } else { " + tmplR[offr:] + " } ; "
				}
			}
			if hasl && iterDepth == 0 {
				if tmplW = " { " + tmplW + " } "; !ustr.Pref(mfr, "v") {
					tmplR = " { " + tmplR + " } "
				}
			}
		} else if len(taggedUnion) > 0 {
			// TAGGED INTERFACE
			if numIndir > 0 {
				tmplR += " var " + ustr.Drop(mfr, ':') + " " + typeName + " ; "
			}

			tmplR += "{ t" + " := data[p] ; p++ ; switch t" + " {"
			tmplW += "{ switch t" + " := " + mfwd + ".(type) {"
			for ti, tu := range taggedUnion {
				tr, _ := genForFieldOrVarOfNamedTypeRW(nf, "", tds, tu, "u", 0, iterDepth, nil)
				tmplR += "\n\t\tcase " + s(ti+1) + ":\n\t\t\t" + "var u " + tu + " ; " + tr + " ; " + ustr.Drop(mfr, ':') + "= u"
				_, tw := genForFieldOrVarOfNamedTypeRW(nf, "t", tds, tu, "", 0, iterDepth, nil)
				tmplW += "\n\t\tcase " + tu + ":\n\t\t\tb·writeB(" + s(ti+1) + ") ; " + tw
			}
			tmplR += "\n\t\tdefault:\n\t\t\t" + ustr.Drop(mfr, ':') + "= nil"
			if optIgnoreUnknownTypeCases {
				tmplW += "\n\t\tdefault:\n\t\t\tb·writeB(0)"
			} else {
				if ustr.Pref(mfw, "me.") {
					mfw = ", " + mfw[3:] + " field"
				} else {
					mfw = ""
				}
				tmplW += "\n\t\tcase nil:\n\t\t\tb·writeB(0)" + "\n\t\tdefault:\n\t\t\tpanic(\"" + tds.TName + ".marshalTo" + mfw + ": while attempting to serialize a non-nil " + typeName + ", encountered a concrete type not mentioned in your corresponding tagged-union field-tag\")\n\t\t\t// panic(fmt.Sprintf(\"%T\", t)) // don't want fmt in by default, but it's here to uncomment when the temporary need arises"
			}
			tmplR += "\n\t}}"
			tmplW += "\n\t}}"
		} else {
			// OTHER

			if fs := fixedSizeForTypeSpec(typeName); fs > 0 && !optNoFixedSizeCode {
				ensureImportFor(typeName)
				tmplR = mfr + "= *((*" + typeName + ")(unsafe.Pointer(&data[p]))) ; p += " + s(fs)
				if mfwd[0] == '*' {
					mfwd = mfwd[1:]
				} else if ustr.Pref(mfwd, "(*") && ustr.Suff(mfwd, ")") {
					mfwd = mfwd[:len(mfwd)-1][2:]
				} else if ustr.Pref(mfwd, "(*") && ustr.Suff(mfwd, ")[0]") {
					mfwd = mfwd[:len(mfwd)-4][2:]
				} else {
					mfwd = "&" + mfwd
				}
				tmplW = "b·writeN((*[" + s(fs) + "]byte)(unsafe.Pointer(" + mfwd + "))[:])"
			} else if tsyn := typeSyns[typeName]; tsyn != "" {
				tmplR, tmplW = genForFieldOrVarOfNamedTypeRW(fieldName, altNoMe, tds, tsyn, "", numIndir, iterDepth, taggedUnion)
				tmplR = ustr.Replace(tmplR, "(*"+tsyn+")(unsafe.Pointer(", "(*"+typeName+")(unsafe.Pointer(")
			} else {
				var trpref string
				if ustr.Pref(mfr, "v") {
					ensureImportFor(typeName)
					trpref = mfr + "= " + typeName + "{} ; "
				}
				tmplR = "{ " + genLenR(nfr) + " ; if " + lf + " > 0 { if err = " + ustr.Drop(mfr, ':') + ".UnmarshalBinary(data[p : p+" + lf + "]); err != nil { return } ; p += " + lf + " } }"
				tmplW = "{ d, e := " + mfw + ".MarshalBinary() ; if err = e; err != nil { return } ; " + lf + " := " + cast + "(len(d)) ; " + genLenW(nfr) + " ; b·writeN(d) }"

				var islocalserializablestruct bool
				for tspec := range typeDefs {
					if islocalserializablestruct = tspec.Name.Name == typeName; islocalserializablestruct {
						if ustr.Pref(mfw, "(") && ustr.Suff(mfw, ")") && mfw[1] == '*' {
							mfw = ustr.Skip(mfw[1:len(mfw)-1], '*')
						}
						tmplR = "if err = " + ustr.Drop(mfr, ':') + ".unmarshalFrom(&p, data); err != nil { return } ; "
						tmplW = "if err = " + mfw + ".marshalTo(buf); err != nil { return } ; "
						break
					}
				}
				tmplR = trpref + tmplR
				if wk := typeName + "·" + nf; (!islocalserializablestruct) && ustr.IdxB(typeName, '.') < 0 && !typeWarned[wk] {
					typeWarned[wk] = true
					println("take note: in-package type `" + typeName + "` will be (de)serialized (via `" + nf + "`) by `MarshalBinary`/`UnmarshalBinary` (instead of `marshalTo`/`unmarshalFrom`) because `gogen-dump` has not been instructed to generate serialization code for it. if this will compile at all, it will likely still not furnish the intended outcome.")
				}
			}
		}
	}
	return
}

func genSizedR(mfr string, typeName string, byteSize string) string {
	if byteSize == "int64" || byteSize == "uint64" {
		if optSafeVarints {
			return mfr + "= " + typeName + "(*((*" + byteSize + ")(unsafe.Pointer(&data[p])))) ; p += 8"
		}
		byteSize = "8"
	}
	ensureImportFor(typeName)
	return mfr + "= *((*" + typeName + ")(unsafe.Pointer(&data[p]))) ; p += " + byteSize
}

func genSizedW(fieldName string, mfw string, byteSize string) string {
	if byteSize == "int64" || byteSize == "uint64" {
		if optSafeVarints {
			return byteSize + "_" + fieldName + " := " + byteSize + "(" + mfw + ") ; b·writeN(((*[8]byte)(unsafe.Pointer(&" + byteSize + "_" + fieldName + ")))[:])"
		}
		byteSize = "8"
	}
	if ustr.Pref(mfw, "(*") && ustr.Suff(mfw, ")") {
		return "b·writeN(((*[" + byteSize + "]byte)(unsafe.Pointer(" + mfw[2:len(mfw)-1] + ")))[:])"
	} else if ustr.Pref(mfw, "(*") && ustr.Suff(mfw, ")[0]") {
		return "b·writeN(((*[" + byteSize + "]byte)(unsafe.Pointer(" + mfw[2:len(mfw)-4] + ")))[:])"
	} else if mfw[0] == '*' {
		return "b·writeN(((*[" + byteSize + "]byte)(unsafe.Pointer(" + mfw[1:] + ")))[:])"
	}
	return "b·writeN(((*[" + byteSize + "]byte)(unsafe.Pointer(&(" + mfw + "))))[:])"
}

func genLenR(fieldName string) string {
	if optSafeVarints {
		return "l" + fieldName + " := int(*((*uint64)(unsafe.Pointer(&data[p])))) ; p += 8"
	}
	return "l" + fieldName + " := (*((*int)(unsafe.Pointer(&data[p])))) ; p += 8"
}

func genLenW(fieldName string) string {
	return "b·writeN((*[8]byte)(unsafe.Pointer(&l" + fieldName + "))[:])"
}
