package warden

import (
	"maps"
	"slices"
	"strings"
)

type Error string

func (e Error) Error() string { return string(e) }

type Errors map[string][]error

func (e *Errors) Add(key string, err error) {
	if err == nil {
		return
	}
	if *e == nil {
		*e = make(map[string][]error)
	}
	(*e)[key] = append((*e)[key], err)
}

func (e Errors) AsError() error {
	if len(e) > 0 {
		return e
	}
	return nil
}

func (e Errors) Error() string {
	var s strings.Builder
	for i, key := range slices.Sorted(maps.Keys(e)) {
		if i > 0 {
			s.WriteString("; ")
		}

		s.WriteString(key)
		s.WriteString(": [")
		for i, err := range e[key] {
			if i > 0 {
				s.WriteString("; ")
			}
			s.WriteString(err.Error())
		}
		s.WriteString("]")
	}
	return s.String()
}
