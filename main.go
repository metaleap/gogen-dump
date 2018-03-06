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

var (
	genFileName  = "@serializers.gen.go"
	tdot         = tmplDotFile{ProgHint: "github.com/metaleap/gogen-dump", Imps: map[string]string{}}
	goPkgDirPath = tdot.ProgHint
	typeNames    = []string{"fixed", "testStruct", "embName"}
	ts           = map[*ast.TypeSpec]*ast.StructType{}
	tSynonyms    = map[string]string{}

	// if false: varints are read-from/written-to directly but occupy 8 bytes in the stream.
	// if true: also occupy 8 bytes in stream, but expressly converted from/to uint64/int64 as applicable
	optSafeVarints = false // set to true by presence of a command-line arg -safeVarints

	optVarintsInFixedSizeds = false // set to true by presence of command-line arg -varintsInFixedSizeds
)

func main() {
	if len(os.Args) > 1 {
		if typeNames, goPkgDirPath = nil, os.Args[1]; len(os.Args) > 2 {
			if ustr.Suff(os.Args[2], ".go") {
				typeNames, genFileName = os.Args[3:], os.Args[2]
			} else {
				typeNames = os.Args[2:]
			}

			// any flags?
			for i, ditch := 0, false; i < len(typeNames); i++ {
				switch typeNames[i] {
				case "-safeVarints":
					optVarintsInFixedSizeds, ditch = true, true
				case "-varintsInFixedSizeds":
					optSafeVarints, ditch = true, true
				}
				if ditch {
					typeNames = append(typeNames[:i], typeNames[i+1:]...)
				}
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
						} else if tident, _ := t.Type.(*ast.Ident); tident != nil {
							tSynonyms[t.Name.Name] = tident.Name
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
