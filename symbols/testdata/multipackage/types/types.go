package types

// Config is a configuration struct from another package
type Config struct {
	Name  string
	Value int
}

// Logger is a logger struct from another package
type Logger struct {
	Level string
}

// Info logs an info message
func (l Logger) Info(msg string) string {
	return l.Level + ": " + msg
}
