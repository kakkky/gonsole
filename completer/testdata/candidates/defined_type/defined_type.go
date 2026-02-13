package definedtype

// MyInt is a defined type based on int
type MyInt int

// MyString is a defined type based on string
type MyString string

// MySlice is a defined type based on slice
type MySlice []string

// MyMap is a defined type based on map
type MyMap map[string]int

// Config is a struct type
type Config struct {
	Name string
}

// MyPtr is a pointer to struct
type MyPtr *Config

// MyIntPtr is a pointer to int
type MyIntPtr *int

// MyFunc is a function type
type MyFunc func(int) string
