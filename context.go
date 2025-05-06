package warden

import (
	"go/types"
	"strings"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
	"golang.org/x/tools/go/packages"
)

type Context struct {
	StructName string
	Pkg        *packages.Package
	pkgs       []*packages.Package
	statics    []*j.Statement
}

func (c *Context) AddStatic(stmt *j.Statement) {
	c.statics = append(c.statics, stmt)
}

func (c *Context) findObject(rawIdent string) (types.Object, bool, error) {
	var local bool
	var path, ident string
	if dotIdx := strings.LastIndexByte(rawIdent, '.'); dotIdx == -1 {
		ident = rawIdent
		local = true
		path = c.Pkg.PkgPath
	} else {
		path, ident = rawIdent[:dotIdx], rawIdent[dotIdx+1:]
	}

	for _, pkg := range c.pkgs {
		pkg = func(pkg *packages.Package) *packages.Package {
			if pkg.PkgPath == path {
				return pkg
			}
			for importPath, pkg := range pkg.Imports {
				if importPath == path {
					return pkg
				}
			}
			return nil
		}(pkg)
		if pkg == nil {
			continue
		}

		obj := pkg.Types.Scope().Lookup(ident)
		if obj == nil {
			break
		}
		return obj, local, nil
	}

	return nil, false, errors.Errorf("identifier %s not found", rawIdent)
}
