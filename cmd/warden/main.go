package main

import (
	"flag"
	"log"

	"golang.org/x/tools/go/packages"

	"github.com/egsam98/warden"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var tag stringPtr
	flag.Var(&tag, "tag", "Struct tag to represent field name")
	flag.Parse()

	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax | packages.LoadFiles}, flag.Args()...)
	if err != nil {
		return err
	}
	return warden.Parse(pkgs, tag.value)
}

type stringPtr struct {
	value *string
}

func (p *stringPtr) String() string {
	if p.value != nil {
		return *p.value
	}
	return ""
}

func (p *stringPtr) Set(s string) error {
	p.value = &s
	return nil
}
