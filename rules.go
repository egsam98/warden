package warden

import (
	"go/types"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

const errorsPkg = "github.com/egsam98/errors"

var rules = map[string]Rule{
	"required": Required(),
	"default":  Default(),
	"url":      URL(),
	"oneof":    OneOf(),
	"regex":    Regex(),
	"length":   Length(),
	"iso-4217": ISO4217(),
	"nested":   Nested(),
	"custom":   Custom(),
}

var ifaceStringer = ImportStdInterface("fmt", "Stringer")
var ifaceIsZero = types.NewInterfaceType([]*types.Func{
	types.NewFunc(
		0,
		nil,
		"IsZero",
		types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Bool])), false),
	),
}, nil)

type Rule struct {
	SkipNil bool
	Do      func(ctx *Context, field Field, props Properties) (*j.Statement, error)
}

func (r *Rule) Render(
	ctx *Context,
	field Field,
	props Properties,
) (*j.Statement, error) {
	ptr, isPtr := field.Type.(*types.Pointer)
	if isPtr && r.SkipNil {
		field.Type = ptr.Elem()
		field.Deref = true
	}
	stmt, err := r.Do(ctx, field, props)
	if err != nil {
		return nil, err
	}
	if isPtr && r.SkipNil {
		stmt = j.If(field.Gen(false).Op("!=").Nil()).Block(stmt)
	}
	return stmt, nil
}

func Required() Rule {
	return Rule{
		SkipNil: false,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}
			retErr, err := ReturnErr(props, "required")
			if err != nil {
				return nil, err
			}
			return j.If(ifFieldZero(ctx, field)).Block(retErr), nil
		},
	}
}

func Default() Rule {
	return Rule{
		SkipNil: false,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			f := field.Gen()
			return j.If(ifFieldZero(ctx, field)).BlockFunc(func(g *j.Group) {
				fieldPtr, isFieldPtr := field.Type.(*types.Pointer)
				_, isValuePtr := props.Value.Type().(*types.Pointer)
				switch {
				case isFieldPtr && !isValuePtr:
					g.Add(f).Op("=").New(j.Id(fieldPtr.Elem().String()))
					g.Op("*").Add(f).Op("=").Add(props.Value.Gen())
				case !isFieldPtr && isValuePtr:
					g.Add(f).Op("=").Op("*").Add(props.Value.Gen())
				default:
					g.Add(f).Op("=").Add(props.Value.Gen())
				}
			}), nil
		},
	}
}

func ifFieldZero(ctx *Context, field Field) *j.Statement {
	if _, isPtr := field.Type.(*types.Pointer); !isPtr && Implements(field.Type, ifaceIsZero) {
		return field.Gen().Dot(ifaceIsZero.Method(0).Name()).Call()
	} else {
		return field.Gen().Op("==").Add(ZeroValue(ctx, field.Type))
	}
}

func URL() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}

			retErr, err := ReturnErr(props, "must be URL")
			if err != nil {
				return nil, err
			}
			return j.If(j.Id("_").Op(",").Err().Op(":=").Qual("net/url", "Parse").Call(field.GenString())).
				Op(";").Err().Op("!=").Nil().
				Block(retErr), nil
		},
	}
}

func OneOf() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			retErr, err := ReturnErr(props, "must be one of %v", props.Value)
			if err != nil {
				return nil, err
			}
			return j.If(j.Op("!").
				Qual("slices", "Contains").
				Call(props.Value.Gen(), field.Gen())).
				Block(retErr), nil
		},
	}
}

func Regex() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			lit, ok := props.Value.(*Lit)
			if !ok {
				return nil, errors.New("value must be string")
			}
			regex, ok := lit.any.(string)
			if !ok {
				return nil, errors.New("value must be string")
			}

			regexID := j.Id("regex" + ctx.StructName + field.Name)
			ctx.AddStatic(
				j.Var().Add(regexID).Op("=").Qual("regexp", "MustCompile").Call(j.Lit(regex)),
			)
			retErr, err := ReturnErr(props, "must match regex "+regex)
			if err != nil {
				return nil, err
			}
			return j.If(j.Op("!").
				Add(regexID).
				Dot("MatchString").
				Call(field.GenString())).
				Block(retErr), nil
		},
	}
}

func Length() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if props.Value != nil {
				retErr, err := ReturnErr(props, "must have length: %s", props.Value)
				if err != nil {
					return nil, err
				}
				return j.If(j.Len(field.Gen()).Op("==").Add(props.Value.Gen())).Block(retErr), nil
			}

			var stmt = j.Null()
			minimum, hasMin := props.Other["min"]
			if hasMin {
				retErr, err := ReturnErr(props, "must have length %v min", minimum)
				if err != nil {
					return nil, err
				}
				stmt.Add(
					j.If(j.Len(field.Gen()).Op("<").Add(minimum.Gen())).Block(retErr),
				)
			}
			maximum, hasMax := props.Other["max"]
			if hasMax {
				if hasMin {
					stmt.Line()
				}
				retErr, err := ReturnErr(props, "must have length %v max", maximum)
				if err != nil {
					return nil, err
				}
				stmt.If(j.Len(field.Gen()).Op(">").Add(maximum.Gen())).Block(retErr)
			}

			if !hasMin && !hasMax {
				return nil, errors.New("specify either value or min/max pair")
			}
			return stmt, nil
		},
	}
}

func ISO4217() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}
			retErr, err := ReturnErr(props, "must be ISO4217 currency")
			if err != nil {
				return nil, err
			}
			stmt := j.If(
				j.Id("code, _").
					Op(":=").
					Qual("github.com/rmg/iso4217", "ByName").
					Call(field.GenString()).Op(";").Id("code").Op("==").Lit(0),
			).Block(retErr)
			return stmt, nil
		},
	}
}

func Nested() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}
			return j.If(
				j.Err().Op(":=").Add(field.Gen(false)).Dot("Validate").Call().Op(";").Err().Op("!=").Nil(),
			).Block(j.Return(j.Err())), nil
		},
	}
}

func Custom() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			funcId, ok := props.Value.(*Id)
			if !ok {
				return nil, errors.New("value must be identifier")
			}
			funcType, ok := funcId.Object.(*types.Func)
			if !ok {
				return nil, errors.New("value must be func identifier")
			}

			var stmt *j.Statement
			sig := funcType.Signature()
			if sig.Recv() != nil {
				stmt = field.Gen().Dot(funcId.Name()).Call()
			} else {
				firstParam := sig.Params().At(0)
				if firstParam == nil {
					return nil, errors.Errorf("function %s has no parameters", funcType)
				}

				stmt = funcId.Gen().CallFunc(func(g *j.Group) {
					_, isPtr := firstParam.Type().(*types.Pointer)
					switch {
					case field.Deref && isPtr, !field.Deref && !isPtr:
						g.Add(field.Gen(false))
					case !field.Deref && isPtr:
						g.Op("&").Add(field.Gen(false))
					default:
						g.Add(field.Gen())
					}
				})
			}

			return j.If(
				j.Err().Op(":=").Add(stmt).Op(";").Err().Op("!=").Nil(),
			).Block(j.Return(j.Err())), nil
		},
	}
}
