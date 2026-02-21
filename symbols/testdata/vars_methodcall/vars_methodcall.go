package varsmethodcall

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

// GlobalLogger is a top-level variable
var GlobalLogger = NewLogger()

// ResultFromMethod is initialized by a method call on a top-level variable
var ResultFromMethod = GlobalLogger.Info("test")
