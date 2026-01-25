package executor

import "go/token"

//go:generate mockgen -package=executor -source=./fset_provider.go -destination=./fset_provider_mock.go
type fsetProvider interface {
	fset() *token.FileSet
}

type defaultFsetProvider struct{}

func newDefaultFsetProvider() *defaultFsetProvider {
	return &defaultFsetProvider{}
}

func (dfp *defaultFsetProvider) fset() *token.FileSet {
	return token.NewFileSet()
}
