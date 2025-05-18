package warden

import (
	"go/importer"
	"go/types"
	"math/rand/v2"
	"strings"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

func ReturnErr(field Field, props Properties, format string, args ...Property) *j.Statement {
	var stmtArgs []j.Code
	if props.Error != nil {
		stmtArgs = append(stmtArgs, j.Lit(*props.Error))
	} else {
		stmtArgs = append(stmtArgs, j.Lit(format))
		for _, arg := range args {
			stmtArgs = append(stmtArgs, arg.Gen())
		}
	}

	var errStmt j.Code
	if len(stmtArgs) == 1 {
		errStmt = stmtArgs[0]
	} else {
		errStmt = j.Qual("fmt", "Sprintf").Call(stmtArgs...)
	}

	return j.Id("errs").Dot("Add").Call(
		field.Name,
		j.Qual(mod, "Error").Parens(errStmt),
	)
}

func Implements(typ types.Type, iface *types.Interface) bool {
	if _, ok := typ.(*types.Pointer); !ok {
		typ = types.NewPointer(typ)
	}
	return types.Implements(typ, iface)
}

func ZeroValue(ctx *Context, typ types.Type) (*j.Statement, error) {
	switch typ := typ.(type) {
	case *types.Basic:
		switch kind := typ.Kind(); kind {
		case types.Bool, types.UntypedBool:
			return j.False(), nil
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64, types.Uint, types.Uint8, types.Uint16,
			types.Uint32, types.Uint64, types.Float32, types.Float64, types.Complex64, types.Complex128, types.Uintptr:
			return j.Lit(0), nil
		case types.String:
			return j.Lit(""), nil
		case types.UnsafePointer:
			return j.Nil(), nil
		default:
			return nil, errors.Errorf("zeroValue: unexpected basic type kind: %d", kind)
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Interface:
		return j.Nil(), nil
	case *types.Array, *types.Struct:
		return j.Id(typ.String()).Values(), nil
	case namedOrAlias:
		under := typ.Underlying()
		if _, ok := under.(*types.Struct); ok {
			obj := typ.Obj()
			path := obj.Pkg().Path()
			if path == ctx.pkg.PkgPath {
				path = ""
			}
			return j.Parens(j.Qual(path, obj.Name()).Values()), nil
		}
		return ZeroValue(ctx, under)
	default:
		return nil, errors.Errorf("zeroValue: unexpected type: %T", typ)
	}
}

func ImportStdInterface(path, name string) *types.Interface {
	pkg, err := importer.Default().Import(path)
	if err != nil {
		panic(err)
	}
	obj := pkg.Scope().Lookup(name)
	if obj == nil {
		panic(errors.Errorf("no interface %s.%s", path, name))
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		panic(errors.Errorf("no interface %s.%s", path, name))
	}
	return iface
}

func LinesFunc(f func(*j.Group)) *j.Statement {
	return j.CustomFunc(j.Options{Separator: "\n"}, f)
}

func randString() string {
	var s strings.Builder
	for range 5 {
		s.WriteRune(rune(rand.N(122-97) + 97))
	}
	return s.String()
}
