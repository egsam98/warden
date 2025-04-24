package _example

import (
	"warden/another"
)

const One = "one"

type Data struct {
	// [warden]
	// regex = {value = "(.).,(.*)$"}
	A *another.Another
	// [warden]
	// required = true
	// custom = { value = "validateB", method = false }
	B *int
	// [warden]
	// required = true
	// url = true
	// oneof = ["id:warden/another.One", "two", "three"]
	C string
	// [warden]
	// length = { min = "id:warden/another.Allo", max = 34 }
	// [each]
	// url = true
	Arr []string
	// [warden]
	// nested = true
	Nested *Nested
}

type Nested struct {
	// [warden]
	// default = "allo da"
	A *string
}

func validateB(b int) error {
	return nil
}

type Data2 struct {
	Sometinh string
}
