package warden

import (
	"go/types"

	"github.com/egsam98/errors"
	"golang.org/x/tools/go/packages"
)

type Context struct {
	StructName string
	pkgs       []*packages.Package
	currPkg    *packages.Package
}

func (c *Context) FindObject(def string) (types.Object, error) {
	path, ident := ParseDef(def)
	if path == "" {
		path = c.currPkg.PkgPath
	}

	for _, pkg := range c.pkgs {
		if pkg.PkgPath != path {
			continue
		}
		obj := pkg.Types.Scope().Lookup(ident)
		if obj == nil {
			break
		}
		return obj, nil
	}
	return nil, errors.Errorf("definition %s not found", def)
}
