package complex

import subcomplex "github.com/kakkky/gonsole/completer/testdata/complex/sub_complex"

// ComplexConst is a constant
const (
	// ComplexA is a complex constant A
	ComplexA = "complex value A"
	// ComplexB is a complex constant B
	ComplexB = "complex value B"
)

// Complex variable
var ComplexC, ComplexD = "complex value C", "complex value D"

// ComplexType is a complex type
type ComplexType struct{}

// ComplexMethod is a method for ComplexType
func (c ComplexType) ComplexMethod() {}

func ReturnOtherPackageType() subcomplex.SubComplexType {
	return subcomplex.SubComplexType{}
}
