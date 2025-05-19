package omap

import (
	"cmp"
	"iter"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/samber/lo"
)

type OrderedMap[T any] struct {
	m    map[string]T
	keys []string
}

func FromMap(m map[string]any, tomlKeys []toml.Key) *OrderedMap[any] {
	type indexKey struct {
		Index int
		Value string
	}

	var keys []indexKey
	for k, v := range m {
		keys = append(keys, indexKey{
			Index: slices.IndexFunc(tomlKeys, func(item toml.Key) bool { return item[0] == k }),
			Value: k,
		})

		if v, ok := v.(map[string]any); ok {
			tomlKeys := lo.FilterMap(tomlKeys, func(item toml.Key, index int) (toml.Key, bool) {
				if len(item) > 1 && item[0] == k {
					return item[1:], true
				}
				return nil, false
			})
			m[k] = FromMap(v, tomlKeys)
		}
	}

	slices.SortFunc(keys, func(a, b indexKey) int { return cmp.Compare(a.Index, b.Index) })
	return &OrderedMap[any]{
		m:    m,
		keys: lo.Map(keys, func(item indexKey, _ int) string { return item.Value }),
	}
}

func Decode(src string) (*OrderedMap[any], error) {
	var m map[string]any
	meta, err := toml.Decode(src, &m)
	if err != nil {
		return nil, err
	}
	return FromMap(m, meta.Keys()), nil
}

func (o *OrderedMap[T]) Get(key string) (T, bool) {
	value, ok := o.m[key]
	return value, ok
}

func (o *OrderedMap[T]) Set(key string, value T) {
	o.keys = append(o.keys, key)
	if o.m == nil {
		o.m = make(map[string]T)
	}
	o.m[key] = value
}

func (o *OrderedMap[T]) Del(keys ...string) {
	for _, key := range keys {
		delete(o.m, key)
		o.keys = slices.DeleteFunc(o.keys, func(item string) bool { return item == key })
	}
}

func (o *OrderedMap[T]) Len() int { return len(o.m) }

func (o *OrderedMap[T]) Range() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, key := range o.keys {
			if !yield(key, o.m[key]) {
				return
			}
		}
	}
}
