package methodcall

// Logger is a simple logger type
type Logger struct {
	Name string
}

// Info logs an info message and returns the logged message
func (l Logger) Info(msg string) string {
	return "INFO: " + msg
}

// NewLogger creates a new Logger instance
func NewLogger() Logger {
	return Logger{Name: "default"}
}

// logger is a top-level variable that will be used as a receiver
var logger = NewLogger()

// result is a variable assigned from a method call on a top-level variable
var result = logger.Info("test message")
