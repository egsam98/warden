package _example

import (
	"time"

	"warden/_example/another"
)

const One = "one"

type Data struct {
	// [warden]
	// regex = {value = "(.).,(.*)$"}
	A another.Another
	// [warden]
	// required = true
	// custom = { value = "id:validateB", method = false }
	// oneof = ["id:warden/_example/another.Allo", 2, 3]
	B *int
	// [warden]
	// required = true
	// url = true
	// oneof = ["id:warden/_example/another.One", "two", "three"]
	C string
	// [warden]
	// length = { min = "id:warden/_example/another.Allo", max = 34 }
	// [each]
	// url = true
	Arr []string
	// [warden]
	// required = true
	// nested = true
	Nested Nested
	// [warden]
	// required = true
	Time time.Time
}

type Nested struct {
	// [warden]
	// default = "allo da"
	A string
}

func validateB(b int) error {
	return nil
}

type Data2 struct {
	Sometinh string
}
