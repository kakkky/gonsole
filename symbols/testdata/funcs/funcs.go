package funcs

import "context"

// Add adds two integers and returns the sum
func Add(a, b int) int {
	return a + b
}

// ReturnMultiple returns multiple values
func ReturnMultiple() (int, string) {
	return 42, "hello"
}

// Double doubles the value at the given pointer
func Double(n *int) {
	*n *= 2
}

// FetchWithContext fetches data using a context from another package
func FetchWithContext(ctx context.Context, key string) string {
	return key
}
