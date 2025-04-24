package main

import (
	"flag"
	"log"
	"strings"

	"golang.org/x/tools/go/packages"

	"warden"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	flag.Parse()

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadAllSyntax | packages.LoadFiles,
	}, flag.Args()...)
	if err != nil {
		return err
	}

	for i, pkg := range pkgs {
		for fileIdx := range pkg.Syntax {
			inPath := pkg.CompiledGoFiles[fileIdx]
			if strings.Contains(inPath, "gen") {
				continue
			}

			if err := warden.Parse(pkgs, i, fileIdx); err != nil {
				return err
			}
		}
	}

	return nil
}
