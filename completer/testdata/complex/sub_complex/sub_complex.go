package subcomplex

var (
	SubComplexA = "Subcomplex value A"
	SubComplexB = "Subcomplex value B"
)

var SubComplexC, SubComplexD = "Subcomplex value C", "Subcomplex value D"

type SubComplexType struct{}

func (c SubComplexType) SubComplexMethod() {}
