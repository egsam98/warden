package main

import (
	"flag"
	"log"

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
	pkgs, err := packages.Load(&packages.Config{Mode: packages.LoadAllSyntax | packages.LoadFiles}, flag.Args()...)
	if err != nil {
		return err
	}
	return warden.Parse(pkgs)
}
