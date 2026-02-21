package interfaces

// Reader defines a reading interface
type Reader interface {
	// Read reads data
	Read(p []byte) (n int, err error)
}

// Writer defines a writing interface
type Writer interface {
	// Write writes data
	Write(p []byte) (n int, err error)
}
