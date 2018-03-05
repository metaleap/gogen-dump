package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/loader"
	"os"
	"path/filepath"

	"github.com/go-leap/dev/go"
	"github.com/go-leap/fs"
	"github.com/go-leap/str"
)

const (
	// if false: varints are read-from/written-to directly but occupy 8 bytes in the stream.
	// if true: also occupy 8 bytes in stream, but expressly converted from/to uint64/int64 as applicable
	safeVarInts = false
)

var (
	genFileName  = "@serializers.gen.go"
	tdot         = tmplDotFile{ProgHint: "github.com/metaleap/gogen-dump", Imps: map[string]string{}}
	goPkgDirPath = tdot.ProgHint
	typeNames    = []string{"testStruct", "embName"}
	ts           = map[*ast.TypeSpec]*ast.StructType{}
)

func main() {
	if len(os.Args) > 1 {
		if typeNames, goPkgDirPath = nil, os.Args[1]; len(os.Args) > 2 {
			if ustr.Suff(os.Args[2], ".go") {
				typeNames, genFileName = os.Args[3:], os.Args[2]
			} else {
				typeNames = os.Args[2:]
			}
		}
	}
	if !(ufs.IsDir(goPkgDirPath) || ufs.IsFile(goPkgDirPath)) {
		goPkgDirPath = udevgo.GopathSrc(goPkgDirPath)
	}
	if ufs.IsFile(goPkgDirPath) {
		goPkgDirPath = filepath.Dir(goPkgDirPath)
	}
	goPkgImpPath := udevgo.DirPathToImportPath(goPkgDirPath)

	var gofilepaths []string
	ufs.WalkFilesIn(goPkgDirPath, func(fp string) bool {
		if ustr.Suff(fp, ".go") && fp != genFileName {
			gofilepaths = append(gofilepaths, fp)
		}
		return true
	})

	goast := loader.Config{Cwd: goPkgDirPath}
	goast.CreateFromFilenames(goPkgImpPath, gofilepaths...)
	goprog, err := goast.Load()
	if err != nil {
		panic(err)
	}
	for _, gofile := range goprog.Package(goPkgImpPath).Files {
		tdot.PName = gofile.Name.Name
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
	}

	if collectTypes(); len(tdot.Types) == 0 {
		println("nothing to generate")
	} else {
		genDump()
	}
}

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
				if tdf.typeIdent = typeIdent(fld.Type); tdf.typeIdent == "" {
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

// we go by type spec strings because they can also occur in struct-field-tags for tagged-unions
func typeIdent(t ast.Expr) string {
	if ident, _ := t.(*ast.Ident); ident != nil {
		return ident.Name
	} else if star, _ := t.(*ast.StarExpr); star != nil {
		if tident := typeIdent(star.X); tident != "" {
			return "*" + tident
		}
		return ""
	} else if arr, _ := t.(*ast.ArrayType); arr != nil {
		if tident := typeIdent(arr.Elt); tident != "" {
			if lit, _ := arr.Len.(*ast.BasicLit); lit != nil && lit.Kind == token.INT {
				return "[" + lit.Value + "]" + tident
			}
			return "[]" + tident
		}
		return ""
	} else if ht, _ := t.(*ast.MapType); ht != nil {
		if tidkey := typeIdent(ht.Key); tidkey != "" {
			if tidval := typeIdent(ht.Value); tidval != "" {
				return "map[" + tidkey + "]" + tidval
			}
		}
		return ""
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
		return pkgname + "." + sel.Sel.Name
	} else if iface, _ := t.(*ast.InterfaceType); iface != nil {
		return "interface{}"
	} else if fn, _ := t.(*ast.FuncType); fn != nil {
		return ""
	} else if ch, _ := t.(*ast.ChanType); ch != nil {
		return ""
	}
	panic(fmt.Sprintf("%T", t))
}
