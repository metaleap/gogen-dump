package main

import (
	"go/ast"
	"go/build"
	"go/types"
	"golang.org/x/tools/go/loader"
	"os"
	"path/filepath"
	"time"

	"github.com/go-leap/dev/go"
	"github.com/go-leap/fs"
	"github.com/go-leap/str"
)

var (
	genFileName = "@serializers.gen.go" // this mutable will first be set to cmdline-arg-specified alt .go file-name (if given), then only later expanded to full absolute output file path
	tdot        = tmplDotFile{ProgHint: "github.com/metaleap/gogen-dump", Imps: map[string]*tmplDotPkgImp{}}
	typeNames   = []string{"simWorld", "city", "company", "school", "family", "person", "hobby", "pet", "petPiranha", "petCat", "petDog", "petHamster", "fixedSize"}
	typeDefs    = map[*ast.TypeSpec]*ast.StructType{}
	typeObjs    = map[string]types.Type{}
	typeSizes   types.Sizes
	typeSyns    = map[string]string{ // added to this map will be: any -foo=bar args, plus parsed in-package type synonyms / type aliases
		"time.Duration": "int64",
	}
	typeWarned   = map[string]bool{}
	goPkgDirPath = tdot.ProgHint
	goProg       *loader.Program

	// if false: varints are read-from/written-to directly but occupy 8 bytes in the stream.
	// if true: also occupy 8 bytes in stream, but expressly converted from/to uint64/int64 as applicable
	optSafeVarints = false // set to true by presence of a command-line arg -safeVarints

	optNoFixedSizeCode = false // set to true by presence of command-line arg -noFixedSizeCode

	optVarintsNotFixedSize = false // set to true by presence of command-line arg -varintsNotFixedSize

	optIgnoreUnknownTypeCases = false // set to true by presence of command-line arg -ignoreUnknownTypeCases

	optHeuristicLenStrings = 7

	optHeuriticLenSlices = 2

	optHeuristicLenMaps = 2

	optHeuristicSizeUnknowns = 8

	optFixedSizeMaxSizeInGB = 2 // 1024 = 1TB, up to 1048576 = 1PB — this amount is never really allocated (if not strictly needed) and only matters for the specific case of dynamic-length slices of fixed-size elems, where this describes the theoretically-supported (by generated de/serialization code) upper bound of total RAM cost for the entire slice — whenever exceeded, instead of a single bytes-copy-op a normal per-elem iteration runs
)

func main() {
	timestarted := time.Now()
	if len(os.Args) > 1 {
		if typeNames, goPkgDirPath = nil, os.Args[1]; len(os.Args) > 2 {
			if ustr.Suff(os.Args[2], ".go") {
				typeNames, genFileName = os.Args[3:], os.Args[2]
			} else {
				typeNames = os.Args[2:]
			}

			// any flags?
			for i, ditch := 0, false; i < len(typeNames); i++ {
				if tn := typeNames[i]; tn[0] == '-' && len(tn) > 1 {
					switch tn {
					case "-safeVarints", "--safeVarints":
						optSafeVarints, ditch = true, true
					case "-varintsNotFixedSize", "--varintsNotFixedSize":
						optVarintsNotFixedSize, ditch = true, true
					case "-ignoreUnknownTypeCases", "--ignoreUnknownTypeCases":
						optIgnoreUnknownTypeCases, ditch = true, true
					default:
						if tsyn, tref := ustr.BreakOnFirstOrPref(ustr.Skip(tn, '-'), "="); tsyn != "" && tref != "" {
							ditch, typeSyns[tsyn] = true, tref
						}
					}
				}
				if ditch {
					ditch, typeNames = false, append(typeNames[:i], typeNames[i+1:]...)
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
	if len(gofilepaths) == 0 {
		panic("no .go files found in: " + goPkgDirPath)
	}

	if typeSizes = types.SizesFor(build.Default.Compiler, build.Default.GOARCH); typeSizes == nil && (!optNoFixedSizeCode) {
		optNoFixedSizeCode = true
		println("fixed-size optimizations won't be generated due to lack of `go/types` support for Go compiler `" + build.Default.Compiler + "` — use `-noFixedSizeCode` to not show this message again.")
	}
	goload := loader.Config{Cwd: goPkgDirPath}
	goload.CreateFromFilenames(goPkgImpPath, gofilepaths...)
	var err error
	goProg, err = goload.Load()
	if err != nil {
		panic(err)
	}
	goprogpkg := goProg.Package(goPkgImpPath)
	for _, gofile := range goprogpkg.Files {
		tdot.PName = gofile.Name.Name
		for _, decl := range gofile.Decls {
			if gd, _ := decl.(*ast.GenDecl); gd != nil {
				for _, spec := range gd.Specs {
					if t, _ := spec.(*ast.TypeSpec); t != nil && t.Name != nil {
						if s, _ := t.Type.(*ast.StructType); s != nil {
							if len(typeNames) == 0 || ustr.In(t.Name.Name, typeNames...) {
								typeDefs[t] = s
							}
							typeObjs[t.Name.Name] = goprogpkg.Pkg.Scope().Lookup(t.Name.Name).Type().Underlying()
						} else if tident, _ := t.Type.(*ast.Ident); tident != nil {
							typeSyns[t.Name.Name] = tident.Name
						} else {
							tid, _ := typeIdentAndFixedSize(t.Type)
							typeSyns[t.Name.Name] = tid
						}
					}
				}
			}
		}
	}
	for _, tn := range typeNames {
		found := len(typeSyns[tn]) > 0
		if !found {
			for t := range typeDefs {
				if found = (t.Name.Name == tn); found {
					break
				}
			}
		}
		if !found {
			println(tn + ": type not found")
		}
	}
	if collectTypes(); len(tdot.Structs) == 0 {
		println("nothing to generate")
	} else if err = genDump(); err != nil {
		panic(err)
	} else {
		timetaken := time.Now().Sub(timestarted)
		os.Stdout.WriteString("generated methods for " + s(len(tdot.Structs)) + " structs in " + timetaken.String() + " to:\n" + genFileName + "\n")
	}
}
