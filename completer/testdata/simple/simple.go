package simple

// SimpleConst is a constant
const SimpleConst = "constant value"

// SimpleVar is a variable
var SimpleVar = "variable value"

// SimpleType is a simple type
type SimpleType struct {
	SimpleField string
}

// SimpleMethod is a method for SimpleType
func (s SimpleType) SimpleMethod() string {
	return "method result"
}

// SimpleFunc is a simple function
func SimpleFunc() string {
	return "function result"
}

type SimpleInterface interface {
	// SimpleMethod is a method of SimpleInterface
	SimpleMethod() string
}
