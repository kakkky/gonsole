package main

import "github.com/kakkky/gonsole/completer/testdata/candidates/multipackage/types"

// GetConfig returns a Config from another package
func GetConfig() types.Config {
	return types.Config{Name: "test", Value: 42}
}

// GetLogger returns a Logger from another package
func GetLogger() types.Logger {
	return types.Logger{Level: "INFO"}
}

// Service has a method that returns a type from another package
type Service struct {
	Name string
}

// GetConfigFromMethod returns a Config from another package via method
func (s Service) GetConfigFromMethod() types.Config {
	return types.Config{Name: s.Name, Value: 100}
}
