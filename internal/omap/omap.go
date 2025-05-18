package omap

import (
	"iter"
	"slices"

	"github.com/BurntSushi/toml"
)

type OrderedMap struct {
	m    map[string]any
	keys []toml.Key
}

func Decode(src string) (*OrderedMap, error) {
	var m map[string]any
	meta, err := toml.Decode(src, &m)
	if err != nil {
		return nil, err
	}
	return &OrderedMap{m: m, keys: meta.Keys()}, nil
}

func (o *OrderedMap) Range() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		depth := 1
		keys := slices.Clone(o.keys)
		var stop bool
		for len(keys) > 0 && !stop {
			keys = slices.DeleteFunc(keys, func(key toml.Key) bool {
				if stop {
					return true
				}
				if n := len(key); n == depth {
					if n == 2 {
						var value any = o.m
						for _, part := range key {
							value = value.(map[string]any)[part]
						}
						if !yield(key[len(key)-1], value) {
							stop = true
							return true
						}
					}
					return true
				}
				return false
			})
			depth++
		}
	}
}
