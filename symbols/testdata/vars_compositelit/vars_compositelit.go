package varscompositelit

// Person is a simple struct
type Person struct {
	Name string
	Age  int
}

// SimplePerson is initialized with a composite literal
var SimplePerson = Person{Name: "Alice", Age: 30}

// PersonPtr is initialized with a pointer to composite literal
var PersonPtr = &Person{Name: "Bob", Age: 25}
