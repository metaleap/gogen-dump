package main

import (
	"go/ast"
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
