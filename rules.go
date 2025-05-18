package warden

import (
	"fmt"
	"go/ast"
	"go/types"
	"time"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

var rules = make(map[string]Rule)

func init() {
	rules = map[string]Rule{
		"required":  Required(),
		"default":   Default(),
		"url":       URL(),
		"oneof":     OneOf(),
		"regex":     Regex(),
		"length":    Length(),
		"non-empty": NonEmpty(),
		"iso-4217":  ISO4217(),
		"custom":    Custom(),
		"dive":      Dive(),
	}
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
	SkipNilPtr bool
	Do         func(ctx *Context, field Field, props Properties) (*j.Statement, error)
}

func (r *Rule) Render(
	ctx *Context,
	field Field,
	props Properties,
) (*j.Statement, error) {
	ptr, isPtr := field.Type.(*types.Pointer)
	if isPtr && r.SkipNilPtr {
		field.Deref = true
		field.Type = ptr.Elem()
		field.Expr = field.Expr.(*ast.StarExpr).X
	}
	stmt, err := r.Do(ctx, field, props)
	if err != nil {
		return nil, err
	}
	if isPtr && r.SkipNilPtr {
		stmt = j.If(field.Gen(false).Op("!=").Nil()).Block(stmt)
	}
	return stmt, nil
}

func Dive() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			stmt, err := dive(ctx, field, props, field.Type)
			if err != nil {
				return nil, err
			}
			return j.Id("errs").Dot("Add").Call(field.Name, stmt), nil
		},
	}
}

func dive(ctx *Context, field Field, props Properties, typ types.Type) (*j.Statement, error) {
	switch typ.(type) {
	case *types.Struct:
		structType, ok := field.Expr.(*ast.StructType)
		if !ok {
			return nil, errors.Errorf("expression must be *ast.StructType, got: %T", field.Expr)
		}
		exprs, err := parseStruct(ctx, structType)
		if err != nil {
			return nil, err
		}
		return j.Func().Params().Error().BlockFunc(func(g *j.Group) {
			self := j.Id("self").Op(":=")
			if !field.Deref {
				self.Op("&")
			}
			self.Add(field.Gen(false))

			g.Add(self)
			g.Var().Id("errs").Qual(mod, "Errors")
			for _, expr := range exprs {
				g.Add(expr)
			}
			g.Return(j.Id("errs").Dot("AsError").Call())
		}).Call(), nil
	case *types.Array, *types.Slice, *types.Map:
		innerType := typ.(interface{ Elem() types.Type }).Elem()
		var innerExpr ast.Expr
		switch expr := field.Expr.(type) {
		case *ast.ArrayType:
			innerExpr = expr.Elt
		case *ast.MapType:
			innerExpr = expr.Value
		default:
			return nil, errors.Errorf("unexpected inner expression: %T", expr)
		}

		eachField := Field{
			Self:  false,
			Deref: false,
			Id:    "elem",
			Name:  j.Qual("strconv", "Itoa").Call(j.Id("i")),
			Type:  innerType,
			Expr:  innerExpr,
		}

		var eachExprs []*j.Statement
		for ruleName, prop := range props.Other {
			var props Properties
			switch prop := prop.(type) {
			case *Id, *List:
				props.Value = prop
			case *Lit:
				if err := props.parse(ctx, prop.any); err != nil {
					return nil, err
				}
			}

			rule, ok := rules[ruleName]
			if !ok {
				return nil, errors.Errorf("unknown rule: %q", ruleName)
			}
			expr, err := rule.Render(ctx, eachField, props)
			if err != nil {
				return nil, err
			}
			eachExprs = append(eachExprs, expr)
		}
		return j.Func().Params().Error().Block(
			j.Var().Id("errs").Qual(mod, "Errors"),
			j.For().Id("i, elem").Op(":=").Range().Add(field.Gen()).BlockFunc(func(g *j.Group) {
				for _, expr := range eachExprs {
					g.Add(expr)
				}
			}),
			j.Return(j.Id("errs").Dot("AsError").Call()),
		).Call(), nil
	case *types.Named:
		return field.Gen(false).Dot("Validate").Call(), nil
	case *types.Alias:
		return dive(ctx, field, props, typ.Underlying())
	default:
		return nil, errors.Errorf("unexpected type: %s", typ)
	}
}

func Required() Rule {
	return Rule{
		SkipNilPtr: false,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}
			stmt, err := ifFieldZero(ctx, field)
			return j.If(stmt).Block(
				ReturnErr(field, props, "required"),
			), err
		},
	}
}

func Default() Rule {
	return Rule{
		SkipNilPtr: false,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			f := field.Gen()
			stmt, err := ifFieldZero(ctx, field)
			if err != nil {
				return nil, err
			}

			blockStmt, err := func() (*j.Statement, error) {
				// Call time.ParseDuration for field type time.Duration and string value
				if named, ok := field.Type.(*types.Named); ok && named.String() == "time.Duration" {
					if basic, ok := props.Value.Type().Underlying().(*types.Basic); ok && basic.Kind() == types.String {
						switch prop := props.Value.(type) {
						case *Lit:
							value := prop.any.(string)
							dur, err := time.ParseDuration(value)
							if err != nil {
								return nil, err
							}
							return f.Op("=").Lit(int(dur)).Comment(value), nil
						case *Id:
							return LinesFunc(func(g *j.Group) {
								g.Id("dur, err").Op(":=").Qual("time", "ParseDuration").Call(props.Value.Gen())
								g.If(j.Err().Op("!=").Nil()).Block(j.Return(j.Err()))
								g.Add(f).Op("=").Id("dur")
							}), nil
						default:
							return nil, errors.Errorf("unexpected value's property type: %T", props.Value)
						}
					}
				}

				value := props.Value.Gen()
				fieldPtr, isFieldPtr := field.Type.(*types.Pointer)
				_, isValuePtr := props.Value.Type().(*types.Pointer)
				switch {
				case isFieldPtr && !isValuePtr:
					return LinesFunc(func(g *j.Group) {
						g.Add(f).Op("=").New(j.Id(fieldPtr.Elem().String()))
						g.Op("*").Add(f).Op("=").Add(value)
					}), nil
				case !isFieldPtr && isValuePtr:
					return f.Op("=").Op("*").Add(value), nil
				default:
					return f.Op("=").Add(value), nil
				}
			}()
			if err != nil {
				return nil, err
			}

			return j.If(stmt).Block(blockStmt), nil
		},
	}
}

func ifFieldZero(ctx *Context, field Field) (*j.Statement, error) {
	if _, isPtr := field.Type.(*types.Pointer); !isPtr && Implements(field.Type, ifaceIsZero) {
		return field.Gen().Dot(ifaceIsZero.Method(0).Name()).Call(), nil
	} else {
		zeroVal, err := ZeroValue(ctx, field.Type)
		return field.Gen().Op("==").Add(zeroVal), err
	}
}

func URL() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}

			return j.If(j.Id("_").Op(",").Err().Op(":=").Qual("net/url", "Parse").Call(field.GenString())).
				Op(";").Err().Op("!=").Nil().
				Block(
					ReturnErr(field, props, "must be URL"),
				), nil
		},
	}
}

func OneOf() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			return j.If(j.Op("!").
				Qual("slices", "Contains").
				Call(props.Value.Gen(), field.Gen())).
				Block(
					ReturnErr(field, props, "must be one of %v", props.Value),
				), nil
		},
	}
}

func Regex() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if props.Value == nil {
				return nil, errors.New("value property is required")
			}
			regexID := j.Id(fmt.Sprintf("regex%s_%s", ctx.StructName, randString()))
			ctx.AddStatic(
				j.Var().Add(regexID).Op("=").Qual("regexp", "MustCompile").Call(props.Value.Gen()),
			)
			return j.If(j.Op("!").
				Add(regexID).
				Dot("MatchString").
				Call(field.GenString())).
				Block(
					ReturnErr(field, props, "must match regex %s", props.Value),
				), nil
		},
	}
}

func Length() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if props.Value != nil {
				return j.If(j.Len(field.Gen()).Op("!=").Add(props.Value.Gen())).Block(
					ReturnErr(field, props, "must have length: %v", props.Value),
				), nil
			}

			return LinesFunc(func(g *j.Group) {
				if minimum, ok := props.Other["min"]; ok {
					g.If(j.Len(field.Gen()).Op("<").Add(minimum.Gen())).Block(
						ReturnErr(field, props, "must have length %v min", minimum),
					)
				}
				if maximum, ok := props.Other["max"]; ok {
					g.If(j.Len(field.Gen()).Op(">").Add(maximum.Gen())).Block(
						ReturnErr(field, props, "must have length %v max", maximum),
					)
				}
			}), nil
		},
	}
}

func NonEmpty() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			return j.If(j.Len(field.Gen()).Op("==").Lit(0)).Block(
				ReturnErr(field, props, "must be non empty"),
			), nil
		},
	}
}

func ISO4217() Rule {
	return Rule{
		SkipNilPtr: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, error) {
			if lit, ok := props.Value.(*Lit); ok && lit.any == false {
				return j.Null(), nil
			}
			return j.If(
				j.Id("code, _").
					Op(":=").
					Qual("github.com/rmg/iso4217", "ByName").
					Call(field.GenString()).Op(";").Id("code").Op("==").Lit(0),
			).Block(
				ReturnErr(field, props, "must be ISO4217 currency"),
			), nil
		},
	}
}

func Custom() Rule {
	return Rule{
		SkipNilPtr: true,
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
			).Block(
				j.Id("errs").Dot("Add").Call(field.Name, j.Err()),
			), nil
		},
	}
}
