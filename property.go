package warden

import (
	"go/types"

	j "github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

type Property interface {
	Gen() *j.Statement
	Type() types.Type
	implProperty()
}

type Id struct {
	types.Object
	local bool
}

func (i *Id) Gen() *j.Statement {
	var path string
	if !i.local {
		path = i.Pkg().Path()
	}
	return j.Qual(path, i.Name())
}

func (*Id) implProperty() {}

type Lit struct{ any }

func (l *Lit) Gen() *j.Statement { return j.Lit(l.any) }

func (l *Lit) Type() types.Type {
	switch l.any.(type) {
	case bool:
		return types.Typ[types.Bool]
	case int:
		return types.Typ[types.Int]
	case float64:
		return types.Typ[types.Float64]
	case string:
		return types.Typ[types.String]
	default:
		return types.NewInterfaceType(nil, nil)
	}
}

func (*Lit) implProperty() {}

type List struct {
	props []Property
	typ   types.Type
}

func (l *List) Gen() *j.Statement {
	values := make([]j.Code, len(l.props))
	for i, prop := range l.props {
		values[i] = prop.Gen()
	}
	return j.Index().Id(l.typ.String()).Values(values...)
}

func (l *List) Type() types.Type { return l.typ }

func (*List) implProperty() {}

type Properties struct {
	Value Property
	Error *string
	Other map[string]Property
}

func (p *Properties) parse(ctx *Context, v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		var err error
		p.Value, err = parseProperty(ctx, v)
		return err
	}

	var err error
	if p.Value, err = parseProperty(ctx, m["value"]); err != nil {
		return err
	}

	switch err := m["error"].(type) {
	case nil:
	case string:
		p.Error = &err
	default:
		return errors.Errorf("error property must be string, got %s", err)
	}

	delete(m, "each")
	delete(m, "value")
	delete(m, "error")
	if len(m) > 0 {
		p.Other = make(map[string]Property, len(m))
		for k, v := range m {
			prop, err := parseProperty(ctx, v)
			if err != nil {
				return err
			}
			p.Other[k] = prop
		}
	}
	return nil
}

func parseProperty(ctx *Context, src any) (Property, error) {
	switch src := src.(type) {
	case nil:
		return nil, nil
	case int64:
		return parseProperty(ctx, int(src))
	case string:
		if match := regexVar.FindStringSubmatch(src); len(match) == 2 {
			obj, local, err := ctx.findObject(match[1])
			if err != nil {
				return nil, err
			}
			return &Id{obj, local}, nil
		}
	case []any:
		props := make([]Property, len(src))
		var typ types.Type
		for i, elem := range src {
			prop, err := parseProperty(ctx, elem)
			if err != nil {
				return nil, err
			}

			var _type types.Type
			switch prop := prop.(type) {
			case *Lit, *Id:
				_type = prop.Type()
			default:
				return nil, errors.Errorf("%T is unsupported as list element", prop)
			}

			if typ != nil && !types.AssignableTo(_type, typ) {
				typ = types.NewInterfaceType(nil, nil)
			} else {
				typ = _type
			}

			props[i] = prop
		}
		return &List{props, typ}, nil
	}

	return &Lit{src}, nil
}
