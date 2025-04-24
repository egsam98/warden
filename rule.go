package warden

import (
	"go/types"

	"github.com/dave/jennifer/jen"
	"github.com/egsam98/errors"
)

var rules = make(map[string]Rule)

func init() {
	rules = map[string]Rule{
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
}

type Properties struct {
	Value any
	Error *string
	Other map[string]any
}

func (p *Properties) UnmarshalTOML(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		p.Value = v
		return nil
	}

	p.Value = m["value"]
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
	p.Other = m
	return nil
}

type Rule struct {
	SkipNil bool
	Do      func(ctx *Context, field Field, props Properties) (stmt *jen.Statement, staticVars *jen.Statement, err error)
}

func (r *Rule) Render(
	ctx *Context,
	field Field,
	props Properties,
) (stmt *jen.Statement, staticVars *jen.Statement, err error) {
	ptr, isPtr := field.Type.(*types.Pointer)
	if isPtr && r.SkipNil {
		field.Type = ptr.Elem()
		field.Deref = true
	}
	if stmt, staticVars, err = r.Do(ctx, field, props); err != nil {
		return
	}
	if isPtr && r.SkipNil {
		stmt = jen.If(field.Jen(false).Op("!=").Nil()).Block(stmt)
	}
	return
}

func Get(name string) (*Rule, error) {
	r, ok := rules[name]
	if !ok {
		return nil, errors.Errorf("unknown rule: %q", name)
	}
	return &r, nil
}
