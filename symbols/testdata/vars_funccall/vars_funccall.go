package varsfunccall

// Config is a configuration struct
type Config struct {
	Name string
}

// NewConfig creates a new Config instance
func NewConfig() Config {
	return Config{Name: "default"}
}

// ConfigVar is initialized by a function call
var ConfigVar = NewConfig()
