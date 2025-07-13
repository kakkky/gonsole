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
	typeSet struct {
		name        string
		description string
	}
)

type candidates struct {
	pkgs    []pkgName
	funcs   map[pkgName][]funcSet
	methods map[pkgName][]methodSet
	vars    map[pkgName][]varSet
	consts  map[pkgName][]constSet
	types   map[pkgName][]typeSet
}

func GenerateCandidates(path string) (*candidates, error) {
	node, err := analyze(path)
	if err != nil {
		return nil, err
	}
	cs := convertFromNodeToCandidates(node)
	return &cs, nil
}
