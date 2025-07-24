package completer

type DeclVarRecord struct {
	Pkg   string
	Name  string
	Type  string
	IsPtr bool
}

var DeclVarRecords = []DeclVarRecord{}

// TODO: &つきで定義されたものはポインタ型の関数を列挙するようにしたい
func StoreDeclVarRecord(IsPtr bool, pkg, name, typeName string) {
	DeclVarRecords = append(DeclVarRecords, DeclVarRecord{
		Pkg:   pkg,
		Name:  name,
		Type:  typeName,
		IsPtr: IsPtr,
	})
}

func IsStoredReceiver(name string) bool {
	for _, decl := range DeclVarRecords {
		if decl.Name == name {
			return true
		}
	}
	return false
}
