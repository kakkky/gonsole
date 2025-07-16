package subcomplex

var (
	SubComplexA = "Subcomplex value A"
	SubComplexB = "Subcomplex value B"
)

const SubComplexC, SubComplexD = "Subcomplex value C", "Subcomplex value D"

type SubComplexType struct {
	FieldA string
	FieldB string
}

func (c SubComplexType) SubComplexMethod() {}
