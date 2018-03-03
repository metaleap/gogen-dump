package main

import (
	"go/ast"
	"golang.org/x/tools/go/loader"
	"os"
	"path/filepath"
	"text/template"

	"github.com/go-leap/dev/go"
)

var (
	tdot        = tmplDot{ProgHint: "github.com/go-leap/gen/gogen-dump"}
	filePathSrc = udevgo.GopathSrc(tdot.ProgHint, "test-struct.go")
	typeName    = "testStruct"
	t           *ast.TypeSpec
)

func main() {
	if len(os.Args) > 1 {
		if filePathSrc = os.Args[1]; len(os.Args) > 2 {
			typeName = os.Args[2]
		}
	}
	goast := loader.Config{}
	gofile, err := goast.ParseFile(filePathSrc, nil)
	tdot.PkgName = gofile.Name.Name
	if err != nil {
		panic(err)
	}
loop:
	for _, decl := range gofile.Decls {
		if gd, _ := decl.(*ast.GenDecl); gd != nil {
			for _, spec := range gd.Specs {
				if t, _ = spec.(*ast.TypeSpec); t != nil && t.Name != nil && t.Name.Name == typeName {
					break loop
				} else {
					t = nil
				}
			}
		}
	}
	if t == nil {
		panic(typeName + " was not found in Go package " + filePathSrc)
	} else {
		genDump()
	}
}

func genDump() {
	filePathDst := filepath.Join(filepath.Dir(filePathSrc), "ß."+filepath.Base(filePathSrc))
	println(filePathDst)
	tdot.TypeName = t.Name.Name
	println(tdot.PkgName)
	println(tdot.TypeName)

	tmpl := template.New(filePathDst)
	if _, err := tmpl.Parse(tmplPkg); err != nil {
		panic(err)
	} else if err = tmpl.Execute(os.Stdout, &tdot); err != nil {
		panic(err)
	}
}
