package methods

// Counter is a simple counter type
type Counter struct {
	Value int
}

// Increment increments the counter value
func (c *Counter) Increment() int {
	c.Value++
	return c.Value
}

// GetValue returns the current value
func (c Counter) GetValue() int {
	return c.Value
}
