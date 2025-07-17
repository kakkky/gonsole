package completer

type pkgName string

type (
	funcSet struct {
		name        string
		description string
	}
	methodSet struct {
		name             string
		description      string
		receiverTypeName string
	}
	varSet struct {
		name        string
		description string
	}
	constSet struct {
		name        string
		description string
	}
	structSet struct {
		name        string
		fields      []string
		description string
	}
)

type candidates struct {
	pkgs    []pkgName
	funcs   map[pkgName][]funcSet
	methods map[pkgName][]methodSet
	vars    map[pkgName][]varSet
	consts  map[pkgName][]constSet
	structs map[pkgName][]structSet
}
