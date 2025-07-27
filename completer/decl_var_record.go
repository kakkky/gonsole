package completer

import "fmt"

type DeclVarRecord struct {
	Name string
	Pkg  string
	Rhs  Rhs
}

type Rhs struct {
	Struct Struct
	Var    Var
	Func   Func
	Method Method
}

type Struct struct {
	Type  string
	IsPtr bool
}
type Var struct {
	Name string
}

type Func struct {
	Name  string
	Order int
}
type Method struct {
	Name  string
	Order int
}

var DeclVarRecords = []DeclVarRecord{}

func StoreDeclVarRecord[RhsT Var | Func | Struct | Method](pkg, name string, rhs RhsT) {
	switch v := any(rhs).(type) {
	case Var:
		DeclVarRecords = append(DeclVarRecords, DeclVarRecord{
			Pkg:  pkg,
			Name: name,
			Rhs:  Rhs{Var: v},
		})
	case Func:
		DeclVarRecords = append(DeclVarRecords, DeclVarRecord{
			Pkg:  pkg,
			Name: name,
			Rhs:  Rhs{Func: v},
		})
	case Method:
		DeclVarRecords = append(DeclVarRecords, DeclVarRecord{
			Pkg:  pkg,
			Name: name,
			Rhs:  Rhs{Method: v},
		})
	// Assuming Struct is a type that can be stored in Rhs
	case Struct:
		DeclVarRecords = append(DeclVarRecords, DeclVarRecord{
			Pkg:  pkg,
			Name: name,
			Rhs:  Rhs{Struct: v},
		})
	}
	fmt.Println("Stored DeclVarRecord:", DeclVarRecords)
}

func IsStoredReceiver(name string) bool {
	for _, decl := range DeclVarRecords {
		if decl.Name == name {
			return true
		}
	}
	return false
}

func ReceiverTypePkgName(receiverName string) string {
	for _, decl := range DeclVarRecords {
		if decl.Name == receiverName {
			return decl.Pkg
		}
	}
	return ""
}
