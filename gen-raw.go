package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-leap/str"
)

func genDump() {
	for _, tdt := range tdot.Structs {
		if fs := tdt.fixedSize(); fs > 0 {
			tdt.TmplR = "*me = *((*" + tdt.TName + ")(unsafe.Pointer(&data[0])))"
			tdt.TmplW = "buf.Write((*[" + strconv.Itoa(fs) + "]byte)(unsafe.Pointer(me))[:])"
		} else {
			for _, tdf := range tdt.Fields {
				tdf.TmplR, tdf.TmplW = genForFieldOrVarOfNamedTypeRW(tdf.FName, "", tdt, tdf.typeIdent, "", 0, 0, tdf.taggedUnion)
				if tdf.isLast && !ustr.Has(tdf.finalTypeIdent(), "[") { // drop the very-last, thus ineffectual (and hence linter-triggering) assignment to p
					lastp, lastpalt := ustr.Last(tdf.TmplR, " p++ "), ustr.Last(tdf.TmplR, " p +=")
					if lastpalt > lastp {
						lastp = lastpalt
					}
					if lastp > 0 {
						off, offalt, offnope := ustr.Pos(tdf.TmplR[lastp:], ";"), ustr.Pos(tdf.TmplR[lastp:], "\n"), ustr.Pos(tdf.TmplR[lastp:], "}")
						if offalt > 0 && offalt < off {
							off = offalt
						}
						if off < 0 {
							off = len(tdf.TmplR) - lastp
						}
						if offnope < 0 || offnope > off {
							tdf.TmplR = tdf.TmplR[:lastp] + "/*" + tdf.TmplR[lastp:lastp+off] + "*/" + tdf.TmplR[lastp+off:]
						}
					}
				}
			}
		}
	}

	filePathDst := filepath.Join(goPkgDirPath, genFileName)
	file, err := os.Create(filePathDst)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	if err = genViaTmpl(file); err != nil {
		panic(err)
	}
	os.Stdout.WriteString("generated: " + filePathDst + "\n")
}

func genForFieldOrVarOfNamedTypeRW(fieldName string, altNoMe string, tdstd *tmplDotStructTypeDef, typeName string, tmpVarPref string, numIndir int, iterDepth int, taggedUnion []string) (tmplR string, tmplW string) {
	nf, mfw, mfr, lf, nfr := fieldName, "me."+fieldName, "me."+fieldName, "l"+ustr.Replace(fieldName, ".", ""), ustr.Replace(fieldName, ".", "")
	if altNoMe != "" {
		if mfw = altNoMe; iterDepth > 0 {
			if mfr = ustr.TrimR(altNoMe, ":"); !ustr.Suff(mfr, "["+nf+"]") {
				mfr += "[" + nf + "]"
			}
		}
	}
	if numIndir > 0 {
		mfr = "v" + strconv.Itoa(numIndir) + strconv.Itoa(iterDepth) + ":"
		mfw = "(" + ustr.Times("*", numIndir) + mfw + ")"
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

			tmplR = "{ "
			var numindir int
			for _, r := range typeName {
				if r == '*' {
					numindir++
				} else {
					break
				}
			}
			tr, tw := genForFieldOrVarOfNamedTypeRW(nf, altNoMe, tdstd, typeName[numindir:], "", numindir, iterDepth, taggedUnion)
			for i := 0; i < numindir; i++ {
				tmplR += " ; var p" + strconv.Itoa(i) + strconv.Itoa(iterDepth) + " " + ustr.Times("*", numindir-i) + typeName[numindir:] + " ; "
			}

			for i := 0; i < numindir; i++ {
				tmplR += "if p++; data[p-1] != 0 { "
				tmplW += "if " + ustr.Times("*", i) + mfw + " == nil { buf.WriteByte(0) } else { buf.WriteByte(1) ; "
			}
			tmplR += "\n\t\t" + tr + " ; "
			for i := numindir - 1; i >= 0; i-- {
				if i == numindir-1 {
					tmplR += " ; p" + strconv.Itoa(i) + strconv.Itoa(iterDepth) + " = &v" + strconv.Itoa(numindir) + strconv.Itoa(iterDepth) + " ; "
				} else {
					tmplR += " ; p" + strconv.Itoa(i) + strconv.Itoa(iterDepth) + " = &p" + strconv.Itoa(i+1) + strconv.Itoa(iterDepth) + " ; "
				}
				tmplR += "\n\t}\n\t"
			}
			tmplR += " ; " + mfr + " = p0" + strconv.Itoa(iterDepth) + " } "
			tmplW += "\n\t\t" + tw
			tmplW += "\n\t" + ustr.Times("}", numindir)
		} else if ismap, pclose := ustr.Pref(typeName, "map["), ustr.Pos(typeName[1:], "]")+1; pclose > 0 && (typeName[0] == '[' || ismap) {
			// ARRAY / SLICE / MAP

			arrfixedsize, slen := 0, typeName[1:pclose]
			if slen == "" || ismap {
				if slen = "(" + lf + ")"; optSafeVarints {
					slen = "int" + slen
				}
				tmplR += genLenR(nfr) + " ; " + mfr + "= make(" + typeName + ", " + lf + ") ; "
				tmplW += lf + " := " + cast + "(len(" + mfw + ")) ; " + genLenW(nfr) + " ; "
			} else {
				if numIndir > 0 {
					tmplR += mfr + "= " + typeName + "{} ; "
				}
				arrfixedsize = fixedSizeForTypeSpec(typeName)
			}
			valtypespec, idx := typeName[pclose+1:], "i"+strconv.Itoa(iterDepth)

			if ismap {
				mk, mv := "k"+strconv.Itoa(iterDepth), "m"+strconv.Itoa(iterDepth)
				tmplR += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
				tmplW += "for " + mk + ", " + mv + " := range " + mfr + " {"

				tmplR += "\n\t\tvar b" + mk + " " + typeName[4:pclose]
				tmplR += "\n\t\tvar b" + mv + " " + valtypespec
				tr, _ := genForFieldOrVarOfNamedTypeRW(mk, "", tdstd, typeName[4:pclose], "", 0, iterDepth+1, nil)
				tmplR += "\n\t\t" + tr
				tr, _ = genForFieldOrVarOfNamedTypeRW(mv, "", tdstd, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplR += "\n\t\t" + tr
				tmplR += "\n\t\t" + mfr + "[b" + mk + "] = b" + mv
				_, tw := genForFieldOrVarOfNamedTypeRW(mk, mk, tdstd, typeName[4:pclose], "", 0, iterDepth+1, nil)
				tmplW += "\n\t\t" + tw
				_, tw = genForFieldOrVarOfNamedTypeRW(mv, mv, tdstd, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplW += "\n\t\t" + tw

				tmplR += "\n\t}"
				tmplW += "\n\t}"
			} else if afs := strconv.Itoa(arrfixedsize); arrfixedsize > 0 {
				tmplW = genSizedW(nfr, mfw+"[0]", afs)
				tmplR = genSizedR(mfr, typeName, afs)
			} else if fs := fixedSizeForTypeSpec(valtypespec); fs > 0 {
				sfs := strconv.Itoa(fs)
				// mfr = tdstd.addTypeIfIface(mfr, typeName, taggedUnion)
				tmplR += "if " + slen + " > 0 { " +
					" copy(((*[1125899906842623]byte)(unsafe.Pointer(&" + mfr + "[0])))[0:" + sfs + "*" + slen + "], data[p:p+(" + sfs + "*" + slen + ")]) " +
					" ; p += (" + sfs + "*" + slen + ") }"
				tmplW += "if " + slen + " > 0 { " +
					" buf.Write((*[1125899906842623]byte)(unsafe.Pointer(&" + mfw + "[0]))[:" + sfs + "*" + slen + "]) " +
					" }"
			} else {
				tmplR += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
				tmplW += "for " + idx + " := 0; " + idx + " < " + slen + "; " + idx + "++ {"
				// mfr = tdstd.addTypeIfIface(mfr, typeName, taggedUnion)
				tr, _ := genForFieldOrVarOfNamedTypeRW(idx, mfr, tdstd, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplR += "\n\t\t" + tr
				_, tw := genForFieldOrVarOfNamedTypeRW(idx, mfw+"["+idx+"]", tdstd, valtypespec, "", 0, iterDepth+1, taggedUnion)
				tmplW += "\n\t\t" + tw
				tmplR += "\n\t}"
				tmplW += "\n\t}"
			}
		} else if len(taggedUnion) > 0 {
			// TAGGED INTERFACE

			tmplR += "{ t" + " := data[p] ; p++ ; switch t" + " {"
			tmplW += "{ switch t" + " := " + mfw + ".(type) {"
			for ti, tu := range taggedUnion {
				tr, _ := genForFieldOrVarOfNamedTypeRW(nf, "", tdstd, tu, "u", 0, iterDepth, nil)
				tmplR += "\n\t\tcase " + strconv.Itoa(ti+1) + ":\n\t\t\t" + "var u " + tu + " ; " + tr + " ; " + mfr + "= u"
				_, tw := genForFieldOrVarOfNamedTypeRW(nf, "t", tdstd, tu, "", 0, iterDepth, nil)
				tmplW += "\n\t\tcase " + tu + ":\n\t\t\tbuf.WriteByte(" + strconv.Itoa(ti+1) + ") ; " + tw
			}
			tmplR += "\n\t\tdefault:\n\t\t\t" + mfr + "= nil"
			if optIgnoreUnknownTypeCases {
				tmplW += "\n\t\tdefault:\n\t\t\tbuf.WriteByte(0)"
			} else {
				tdot.Imps["fmt"] = "fmt"
				tmplW += "\n\t\tcase nil:\n\t\t\tbuf.WriteByte(0)" + "\n\t\tdefault:\n\t\t\treturn fmt.Errorf(\"" + tdstd.TName + "." + ustr.Until(ustr.TrimPref(altNoMe, "me."), "[") + ": type %T not mentioned in tagged-union field-tag\", t" + ")"
			}
			tmplR += "\n\t}}"
			tmplW += "\n\t}}"
		} else {
			// OTHER

			if fs := fixedSizeForTypeSpec(typeName); fs > 0 {
				tmplR = mfr + "= *((*" + typeName + ")(unsafe.Pointer(&data[p]))) ; p += " + strconv.Itoa(fs)
				if mfw[0] == '*' {
					mfw = mfw[1:]
				} else if ustr.Pref(mfw, "(*") && ustr.Suff(mfw, ")") {
					mfw = mfw[:len(mfw)-1][2:]
				} else {
					mfw = "&" + mfw
				}
				tmplW = "buf.Write((*[" + strconv.Itoa(fs) + "]byte)(unsafe.Pointer(" + mfw + "))[:])"
			} else if tsyn := tSynonyms[typeName]; tsyn != "" {
				tmplR, tmplW = genForFieldOrVarOfNamedTypeRW(fieldName, altNoMe, tdstd, tsyn, "", numIndir, iterDepth, taggedUnion)
				tmplR = ustr.Replace(tmplR, "(*"+tsyn+")(unsafe.Pointer(", "(*"+typeName+")(unsafe.Pointer(")
			} else {
				if ustr.Pref(mfr, "v") {
					tmplR = mfr + "= " + typeName + "{} ; "
				}
				tmplR += genLenR(nfr) + " ; if err = " + ustr.TrimR(mfr, ":") + ".UnmarshalBinary(data[p : p+" + lf + "]); err != nil { return } ; p += " + lf
				tmplW = "{ d, e := " + mfw + ".MarshalBinary() ; if err = e; err != nil { return } ; " + lf + " := " + cast + "(len(d)) ; " + genLenW(nfr) + " ; buf.Write(d) }"

				for tspec := range ts {
					if tspec.Name.Name == typeName {
						tdstd.HasWData = true
						if ustr.Pref(mfw, "(") && ustr.Suff(mfw, ")") && mfw[1] == '*' {
							mfw = ustr.TrimL(mfw[1:len(mfw)-1], "*")
						}
						tmplW = "if err = " + mfw + ".writeTo(&data); err != nil { return } ; " + lf + " := " + cast + "(data.Len()) ; " + genLenW(nfr) + " ; data.WriteTo(buf)"
						break
					}
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
	return mfr + "= *((*" + typeName + ")(unsafe.Pointer(&data[p]))) ; p += " + byteSize
}

func genSizedW(fieldName string, mfw string, byteSize string) (s string) {
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
