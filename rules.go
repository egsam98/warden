package warden

import (
	"fmt"
	"go/types"
	"strings"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

const errorsPkg = "github.com/egsam98/errors"

func Required() Rule {
	return Rule{
		SkipNil: false,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			if props.Value == false {
				return j.Null(), nil, nil
			}
			stmt := j.Block(
				j.Var().Id("zero").Id(field.Type.String()),
				j.If(field.Jen()).Op("==").Id("zero").Block(
					returnErr("required", props),
				),
			)
			return stmt, nil, nil
		},
	}
}

func Default() Rule {
	return Rule{
		SkipNil: false,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			f := field.Jen()
			stmt := j.Block(
				j.Var().Id("zero").Id(field.Type.String()),
				j.If(f).Op("==").Id("zero").BlockFunc(func(g *j.Group) {
					if ptr, ok := field.Type.(*types.Pointer); ok {
						g.Add(f).Op("=").New(j.Id(ptr.Elem().String()))
						g.Op("*").Add(f).Op("=").Lit(props.Value)
						return
					}
					g.Add(f).Op("=").Lit(props.Value)
				}),
			)
			return stmt, nil, nil
		},
	}
}

func URL() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			if props.Value == false {
				return j.Null(), nil, nil
			}

			stmt := j.If(j.Id("_").Op(",").Err().Op(":=").Qual("net/url", "Parse").Call(field.JenString())).
				Op(";").Err().Op("!=").Nil().Block(
				returnErr("must be URL", props),
			)
			return stmt, nil, nil
		},
	}
}

func OneOf() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			variants, ok := props.Value.([]any)
			if !ok {
				return nil, nil, errors.New("value must be array")
			}

			stmt := j.If(j.Op("!").Qual("slices", "Contains").Call(list(variants, field.Type), field.Jen())).Block(
				returnErr(fmt.Sprintf("must be one of %v", variants), props),
			)
			return stmt, nil, nil
		},
	}
}

func Regex() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			regex, ok := props.Value.(string)
			if !ok {
				return nil, nil, errors.New("value must be string")
			}

			regexID := j.Id("regex" + ctx.StructName + field.Name)
			static := j.Var().Add(regexID).Op("=").Qual("regexp", "MustCompile").Call(j.Lit(regex))
			stmt := j.If(j.Op("!").Add(regexID).Dot("MatchString").Call(field.JenString())).Block(
				returnErr("must match regex "+regex, props),
			)
			return stmt, static, nil
		},
	}
}

func Length() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			if props.Value != nil {
				value, ok := props.Value.(int64)
				if !ok {
					return nil, nil, errors.New("value must be integer")
				}
				stmt := j.If(j.Len(field.Jen()).Op("==").Lit(int(value))).Block(
					returnErr(fmt.Sprintf("must have length: %v", props.Value), props),
				)
				return stmt, nil, nil
			}

			var stmt = j.Null()
			minimum, hasMin := props.Other["min"]
			if hasMin {
				if v, ok := minimum.(int64); ok {
					minimum = int(v)
				}
				stmt.Add(
					j.If(j.Len(field.Jen()).Op("<").Lit(minimum)).Block(
						returnErr(fmt.Sprintf("must have length %d min", minimum), props),
					),
				)
			}
			maximum, hasMax := props.Other["max"]
			if hasMax {
				if v, ok := maximum.(int64); ok {
					maximum = int(v)
				}
				if hasMin {
					stmt.Line()
				}
				stmt.If(j.Len(field.Jen()).Op(">").Lit(maximum)).Block(
					returnErr(fmt.Sprintf("must have length %d max", maximum), props),
				)
			}

			if !hasMin && !hasMax {
				return nil, nil, errors.New("specify either value or min/max pair")
			}
			return stmt, nil, nil
		},
	}
}

func ISO4217() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			if props.Value == false {
				return j.Null(), nil, nil
			}
			stmt := j.If(
				j.Id("code, _").
					Op(":=").
					Qual("github.com/rmg/iso4217", "ByName").
					Call(field.JenString()).Op(";").Id("code").Op("==").Lit(0),
			).Block(
				returnErr("must be ISO4217 currency", props),
			)
			return stmt, nil, nil
		},
	}
}

func Nested() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (stmt *j.Statement, staticVars *j.Statement, err error) {
			if props.Value == false {
				return j.Null(), nil, nil
			}
			return j.If(
				j.Err().Op(":=").Add(field.Jen(false)).Dot("Validate").Call().Op(";").Err().Op("!=").Nil(),
			).Block(j.Return(j.Err())), nil, nil
		},
	}
}

func Custom() Rule {
	return Rule{
		SkipNil: true,
		Do: func(ctx *Context, field Field, props Properties) (*j.Statement, *j.Statement, error) {
			funcDef, ok := props.Value.(string)
			if !ok {
				return nil, nil, errors.New("value must be string")
			}

			var stmt *j.Statement
			if props.Other["method"] == true {
				stmt = field.Jen().Dot(funcDef).Call()
			} else {
				obj, err := ctx.FindObject(funcDef)
				if err != nil {
					return nil, nil, err
				}
				f, ok := obj.(*types.Func)
				if !ok {
					return nil, nil, errors.Errorf("definition %s is not function", funcDef)
				}
				firstParam := f.Signature().Params().At(0)
				if firstParam == nil {
					return nil, nil, errors.Errorf("function %s has no parameters", funcDef)
				}
				stmt = qual(funcDef).CallFunc(func(g *j.Group) {
					_, isPtr := firstParam.Type().(*types.Pointer)
					switch {
					case field.Deref && isPtr, !field.Deref && !isPtr:
						g.Add(field.Jen(false))
					case !field.Deref && isPtr:
						g.Op("&").Add(field.Jen(false))
					default:
						g.Add(field.Jen())
					}
				})
			}

			return j.If(
				j.Err().Op(":=").Add(stmt).Op(";").Err().Op("!=").Nil(),
			).Block(j.Return(j.Err())), nil, nil
		},
	}
}

func list(slice []any, typ types.Type) *j.Statement {
	values := make([]j.Code, len(slice))
	for i, elem := range slice {
		values[i] = j.Lit(elem)
	}
	return j.Index().Id(typ.String()).Values(values...)
}

type Field struct {
	Self, Deref bool
	Name        string
	Type        types.Type
	Object      types.Object
}

func (f *Field) Jen(deref ...bool) *j.Statement {
	_deref := true
	if len(deref) > 0 {
		_deref = deref[0]
	}

	var id *j.Statement
	if f.Self {
		id = j.Id("self." + f.Name)
	} else {
		id = j.Id(f.Name)
	}
	if f.Deref && _deref {
		return j.Op("*").Add(id)
	}
	return id
}

func (f *Field) JenString() *j.Statement {
	fieldType := f.Type
	var named bool
	for {
		switch typ := fieldType.(type) {
		case *types.Alias:
			fieldType = typ.Underlying()
		case *types.Named:
			for i := range typ.NumMethods() {
				if strings.HasSuffix(typ.Method(i).String(), ".String() string") {
					return f.Jen(false).Dot("String").Call()
				}
			}
			named = true
			fieldType = typ.Underlying()
		default:
			stmt := f.Jen()
			if named {
				stmt = j.Id(typ.String()).Parens(stmt)
			}
			return stmt
		}
	}
}

func returnErr(msg string, props Properties) *j.Statement {
	var stmt *j.Statement
	if props.Error != nil {
		stmt = j.Id(*props.Error)
	} else {
		stmt = j.Lit(msg)
	}
	return j.Return(j.Qual(errorsPkg, "New").Call(stmt))
}

func qual(def string) *j.Statement {
	path, ident := ParseDef(def)
	if path == "" {
		return j.Id(ident)
	}
	return j.Qual(path, ident)
}

func ParseDef(def string) (path, ident string) {
	dotIdx := strings.LastIndexByte(def, '.')
	if dotIdx == -1 {
		return "", def
	}
	return def[:dotIdx], def[dotIdx+1:]
}
