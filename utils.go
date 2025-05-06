package warden

import (
	"go/importer"
	"go/types"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

func ReturnErr(props Properties, format string, args ...Property) (*j.Statement, error) {
	stmtArgs := []j.Code{j.Lit(format)}
	if props.Error != nil {
		stmtArgs = append(stmtArgs, j.Id(*props.Error))
	} else {
		for _, arg := range args {
			stmtArgs = append(stmtArgs, arg.Gen())
		}
	}
	return j.Return(j.Qual(errorsPkg, "Errorf").Call(stmtArgs...)), nil
}

func Implements(typ types.Type, iface *types.Interface) bool {
	if _, ok := typ.(*types.Pointer); !ok {
		typ = types.NewPointer(typ)
	}
	return types.Implements(typ, iface)
}

func ZeroValue(ctx *Context, typ types.Type) *j.Statement {
	switch typ := typ.(type) {
	case *types.Basic:
		switch kind := typ.Kind(); kind {
		case types.Bool, types.UntypedBool:
			return j.False()
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64, types.Uint, types.Uint8, types.Uint16,
			types.Uint32, types.Uint64, types.Float32, types.Float64, types.Complex64, types.Complex128, types.Uintptr:
			return j.Lit(0)
		case types.String:
			return j.Lit("")
		case types.UnsafePointer:
			return j.Nil()
		default:
			panic(errors.Errorf("zeroValue: unexpected basic type kind: %d", kind))
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Interface:
		return j.Nil()
	case *types.Array, *types.Struct:
		return j.Id(typ.String()).Values()
	case namedOrAlias:
		under := typ.Underlying()
		if _, ok := under.(*types.Struct); ok {
			obj := typ.Obj()
			path := obj.Pkg().Path()
			if path == ctx.Pkg.PkgPath {
				path = ""
			}
			return j.Parens(j.Qual(path, obj.Name()).Values())
		}
		return ZeroValue(ctx, under)
	default:
		panic(errors.Errorf("zeroValue: unexpected type: %T", typ))
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
