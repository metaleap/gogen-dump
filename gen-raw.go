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
	for _, tdt := range tdot.Structs {
		if fs := tdt.fixedSize(); fs > 0 && !optNoFixedSizeCode {
			ensureImportFor(tdt.TName)
			tdt.TmplR = "*me = *((*" + tdt.TName + ")(unsafe.Pointer(&data[p]))) ; p += " + s(fs)
			tdt.TmplW = "buf.Write((*[" + s(fs) + "]byte)(unsafe.Pointer(me))[:])"
		} else {
			for i := 0; i < len(tdt.Fields); i++ {
				tdf := tdt.Fields[i]
				oneop := tdf.fixedsizeExtNumSkip > 0
				if oneop && !optNoFixedSizeCode {
					addr, fse := "&me."+tdf.FName, s(tdf.fixedsizeExt)
					if i += tdf.fixedsizeExtNumSkip; tdf.typeIdent[0] == '[' {
						addr += "[0]"
					}
					tdf.TmplW = "buf.Write( (*[" + fse + "]byte)(unsafe.Pointer(" + addr + ")) [:] )"
					tdf.TmplR = "*( (*[" + fse + "]byte)(unsafe.Pointer(" + addr + ")) ) = *( (*[" + fse + "]byte)(unsafe.Pointer(&data[p])) ) ; p += " + fse + " "
				} else {
					tdf.TmplR, tdf.TmplW = genForFieldOrVarOfNamedTypeRW(tdf.FName, "", tdt, tdf.typeIdent, "", 0, 0, tdf.taggedUnion)
				}
			}
		}
		tdt.InitialBufSize = tdt.sizeHeur("me.")
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
		mfr = "v" + s(numIndir) + s(iterDepth) + ":"
		mfw = "pv" + s(numIndir-1) + s(iterDepth) //  "(" + ustr.Times("*", numIndir) + mfw + ")"
	} else if altNoMe == "" && tmpVarPref != "" {
		mfr = tmpVarPref // + nfr
	} else if numIndir == 0 && iterDepth > 0 && altNoMe == "" && (ustr.Pref(nf, "k") || ustr.Pref(nf, "m")) {
		mfr = "b" + nf
	}
	var cast string
	if optSafeVarints {
		cast = "uint64"
	}
	switch typeName {
	case "bool":
		tmplW = "if " + mfw + " { buf.WriteByte(1) } else { buf.WriteByte(0) }"
		tmplR = mfr + "= (data[p] == 1) ; p++ "
	case "uint8", "byte":
		tmplW = "buf.WriteByte(" + mfw + ")"
		tmplR = mfr + "= data[p] ; p++ "
	case "int8":
		tmplW = genSizedW(nfr, mfw, "1")
		tmplR = genSizedR(mfr, typeName, "1")
	case "string":
		tmplW = lf + " := " + cast + "(len(" + mfw + ")) ; " + genLenW(nfr) + " ; buf.WriteString(" + mfw + ")"
		tmplR = genLenR(nfr) + " ; " + mfr + "= string(data[p : p+" + lf + "]) ; p += " + lf
	case "int16", "uint16":
		tmplW = genSizedW(nfr, mfw, "2")
		tmplR = genSizedR(mfr, typeName, "2")
	case "rune", "int32", "float32", "uint32":
		tmplW = genSizedW(nfr, mfw, "4")
		tmplR = genSizedR(mfr, typeName, "4")
	case "complex64", "float64", "uint64", "int64":
		tmplW = genSizedW(nfr, mfw, "8")
		tmplR = genSizedR(mfr, typeName, "8")
	case "complex128":
		tmplW = genSizedW(nfr, mfw, "16")
		tmplR = genSizedR(mfr, typeName, "16")
	case "uint", "uintptr":
		tmplW = genSizedW(nfr, mfw, "uint64")
		tmplR = genSizedR(mfr, typeName, "uint64")
	case "int":
		tmplW = genSizedW(nfr, mfw, "int64")
		tmplR = genSizedR(mfr, typeName, "int64")
	default:
		if typeName[0] == '*' {
			// POINTER

			numindir := len(typeName) - len(ustr.Skip(typeName, '*'))
			tn := typeName[numindir:]
			tr, tw := genForFieldOrVarOfNamedTypeRW(nf, altNoMe, tds, tn, "", numindir, iterDepth, taggedUnion)
			tmplR = "{ "
			for i := 0; i < numindir; i++ {
				tmplR += " var p" + s(i) + s(iterDepth) + " " + ustr.Times("*", numindir-i) + tn + " ; "
			}
			for i := 0; i < numindir; i++ {
				tmplR += "if p++; data[p-1] != 0 { "
				if i == 0 {
					tmplW += "if " + mfw + " == nil { buf.WriteByte(0) } else { buf.WriteByte(1) ; pv0" + s(iterDepth) + " := *" + mfw + " ; "
				} else {
					tmplW += "if pv" + s(i-1) + s(iterDepth) + " == nil { buf.WriteByte(0) } else { buf.WriteByte(1) ; pv" + s(i) + s(iterDepth) + " := *pv" + s(i-1) + s(iterDepth) + " ; "
				}
			}
			tmplR += "\n\t\t" + tr + " ; "
			for i := numindir - 1; i >= 0; i-- {
				if i == numindir-1 {
					tmplR += " ; p" + s(i) + s(iterDepth) + " = &v" + s(numindir) + s(iterDepth) + " ; "
				} else {
					tmplR += " ; p" + s(i) + s(iterDepth) + " = &p" + s(i+1) + s(iterDepth) + " ; "
				}
				tmplR += "\n\t}\n\t"
			}
			tmplR += " ; " + mfr + " = p0" + s(iterDepth) + " } "
			tmplW += "\n\t\t" + tw
			tmplW += "\n\t" + ustr.Times("}", numindir)
		} else if ismap, pclose := ustr.Pref(typeName, "map["), ustr.IdxBMatching(typeName, ']', '['); pclose > 0 && (typeName[0] == '[' || ismap) {
			// ARRAY / SLICE / MAP

			arrfixedsize, slen := 0, typeName[1:pclose]
			if slen == "" || ismap {
				if slen = "(" + lf + ")"; optSafeVarints {
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
				tmplW += "for " + mk + ", " + mv + " := range " + mfr + " {"

				tmplR += "\n\t\tvar b" + mk + " " + keytypespec
				tmplR += "\n\t\tvar b" + mv + " " + valtypespec
				tr, _ := genForFieldOrVarOfNamedTypeRW(mk, "", tds, keytypespec, "", 0, iterDepth+1, nil)
				tmplR += "\n\t\t" + tr
				tr, _ = genForFieldOrVarOfNamedTypeRW(mv, "", tds, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplR += "\n\t\t" + tr
				tmplR += "\n\t\t" + mfr + "[b" + mk + "] = b" + mv
				_, tw := genForFieldOrVarOfNamedTypeRW(mk, mk, tds, keytypespec, "", 0, iterDepth+1, nil)

				tmplW += "\n\t\t" + tw
				_, tw = genForFieldOrVarOfNamedTypeRW(mv, mv, tds, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplW += "\n\t\t" + tw

				tmplR += "\n\t}"
				tmplW += "\n\t}"
			} else if afs := s(arrfixedsize); arrfixedsize > 0 && !optNoFixedSizeCode {
				tmplW = genSizedW(nfr, mfw+"[0]", afs)
				tmplR = genSizedR(mfr, typeName, afs)
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
						" buf.Write((*[" + s(fixedsizemax-1) + "]byte)(unsafe.Pointer(&" + ustr.Drop(mfw, ':') + "[0]))[:" + sfs + "*" + slen + "]) " +
						" } else { " + tmplW[offw:] + "} ; "
					tmplR = tmplR[:offr] + " ; if " + slen + " > 0 && " + slen + " < " + s(maxlen) + " { " +
						" lmul := " + sfs + "*" + slen + " ; copy(((*[" + s(fixedsizemax-1) + "]byte)(unsafe.Pointer(&" + ustr.Drop(mfr, ':') + "[0])))[0:lmul], data[p:p+(lmul)])  ; p += (lmul) " +
						" } else { " + tmplR[offr:] + " } ; "
				}
			}
		} else if len(taggedUnion) > 0 {
			// TAGGED INTERFACE

			tmplR += "{ t" + " := data[p] ; p++ ; switch t" + " {"
			tmplW += "{ switch t" + " := " + mfw + ".(type) {"
			for ti, tu := range taggedUnion {
				tr, _ := genForFieldOrVarOfNamedTypeRW(nf, "", tds, tu, "u", 0, iterDepth, nil)
				tmplR += "\n\t\tcase " + s(ti+1) + ":\n\t\t\t" + "var u " + tu + " ; " + tr + " ; " + mfr + "= u"
				_, tw := genForFieldOrVarOfNamedTypeRW(nf, "t", tds, tu, "", 0, iterDepth, nil)
				tmplW += "\n\t\tcase " + tu + ":\n\t\t\tbuf.WriteByte(" + s(ti+1) + ") ; " + tw
			}
			tmplR += "\n\t\tdefault:\n\t\t\t" + mfr + "= nil"
			if optIgnoreUnknownTypeCases {
				tmplW += "\n\t\tdefault:\n\t\t\tbuf.WriteByte(0)"
			} else {
				tmplW += "\n\t\tcase nil:\n\t\t\tbuf.WriteByte(0)" + "\n\t\tdefault:\n\t\t\tpanic(\"" + tds.TName + ".marshalTo: while attempting to serialize a non-nil " + typeName + ", encountered a concrete type not mentioned in corresponding tagged-union field-tag\")\n\t\t\t// panic(fmt.Sprintf(\"%T\", t))"
			}
			tmplR += "\n\t}}"
			tmplW += "\n\t}}"
		} else {
			// OTHER

			if fs := fixedSizeForTypeSpec(typeName); fs > 0 && !optNoFixedSizeCode {
				ensureImportFor(typeName)
				tmplR = mfr + "= *((*" + typeName + ")(unsafe.Pointer(&data[p]))) ; p += " + s(fs)
				if mfw[0] == '*' {
					mfw = mfw[1:]
				} else if ustr.Pref(mfw, "(*") && ustr.Suff(mfw, ")") {
					mfw = mfw[:len(mfw)-1][2:]
				} else {
					mfw = "&" + mfw
				}
				tmplW = "buf.Write((*[" + s(fs) + "]byte)(unsafe.Pointer(" + mfw + "))[:])"
			} else if tsyn := typeSyns[typeName]; tsyn != "" {
				tmplR, tmplW = genForFieldOrVarOfNamedTypeRW(fieldName, altNoMe, tds, tsyn, "", numIndir, iterDepth, taggedUnion)
				tmplR = ustr.Replace(tmplR, "(*"+tsyn+")(unsafe.Pointer(", "(*"+typeName+")(unsafe.Pointer(")
			} else {
				var trpref string
				if ustr.Pref(mfr, "v") {
					ensureImportFor(typeName)
					trpref = mfr + "= " + typeName + "{} ; "
				}
				tmplR = genLenR(nfr) + " ; if " + lf + " > 0 { if err = " + ustr.Drop(mfr, ':') + ".UnmarshalBinary(data[p : p+" + lf + "]); err != nil { return } ; p += " + lf + " }"
				tmplW = "{ d, e := " + mfw + ".MarshalBinary() ; if err = e; err != nil { return } ; " + lf + " := " + cast + "(len(d)) ; " + genLenW(nfr) + " ; buf.Write(d) }"

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
			return byteSize + "_" + fieldName + " := " + byteSize + "(" + mfw + ") ; buf.Write(((*[8]byte)(unsafe.Pointer(&" + byteSize + "_" + fieldName + ")))[:])"
		}
		byteSize = "8"
	}
	if ustr.Pref(mfw, "(*") && ustr.Suff(mfw, ")") {
		return "buf.Write(((*[" + byteSize + "]byte)(unsafe.Pointer(" + mfw[2:len(mfw)-1] + ")))[:])"
	} else if mfw[0] == '*' {
		return "buf.Write(((*[" + byteSize + "]byte)(unsafe.Pointer(" + mfw[1:] + ")))[:])"
	}
	return "buf.Write(((*[" + byteSize + "]byte)(unsafe.Pointer(&(" + mfw + "))))[:])"
}

func genLenR(fieldName string) string {
	if optSafeVarints {
		return "l" + fieldName + " := int(*((*uint64)(unsafe.Pointer(&data[p])))) ; p += 8"
	}
	return "l" + fieldName + " := (*((*int)(unsafe.Pointer(&data[p])))) ; p += 8"
}

func genLenW(fieldName string) string {
	return "buf.Write((*[8]byte)(unsafe.Pointer(&l" + fieldName + "))[:])"
}
