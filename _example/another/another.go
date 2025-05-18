package another

type Another string

func (Another) String() string { return "Another" }

const One = "Hello"

var Allo = 100

type Struct struct {
	// [warden]
	// url = true
	Field string
}
