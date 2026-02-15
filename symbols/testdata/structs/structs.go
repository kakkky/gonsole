package structs

// SimpleStruct has simple fields
type SimpleStruct struct {
	FieldA string
	FieldB int
}

// MultiFieldStruct has multiple fields on same line
type MultiFieldStruct struct {
	X, Y, Z int
}

// Base is used for embedding
type Base struct {
	ID int
}

// Derived embeds Base
type Derived struct {
	Base
	Name string
}
