package _example

import (
	"time"

	an "github.com/egsam98/warden/_example/another"
)

const One = "one"

type Data2 struct {
	// [warden]
	// default = "allo da"
	A string
}

type Data struct {
	// [warden]
	// regex = "(.).,(.*)$"
	A an.Another `json:"a"`
	// [warden]
	// required = true
	// custom = { value = "id:validateB", method = false }
	// oneof = ["id:github.com/egsam98/warden/_example/another.Allo", 2, 3]
	B *int `json:"b"`
	// [warden]
	// required = true
	// url = true
	// oneof = ["id:github.com/egsam98/warden/_example/another.One", "two", "three"]
	C string `json:"c,omitempty"`
	// [warden]
	// length = { min = "id:github.com/egsam98/warden/_example/another.Allo", max = 34 }
	// [warden.dive]
	// non-empty = true
	// [warden.dive.dive]
	// regex = "(.).,(.*)$"
	// length = "id:github.com/egsam98/warden/_example/another.Allo"
	// url = { value = true, error = "no url" }
	Arr [][]string `json:"arr"`
	// [warden]
	// [warden.dive]
	// [warden.dive.dive]
	Arr2 []*Data2 `json:"arr2"`
	// [warden]
	// required = true
	// [warden.dive]
	Data2 *an.Struct `json:"data2"`
	// [warden]
	// [warden.dive]
	Data3 struct {
		// [warden]
		// required = true
		Test bool `json:"test"`
	}
	// [warden]
	// required = true
	Time time.Time `json:"time"`
	// [warden]
	// default = "30s"
	Duration time.Duration
}

func validateB(b int) error {
	return nil
}
