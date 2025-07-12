package completer

type pkgName string
type funcName string
type varName string
type constName string
type typeName string
type methodName string

type receiverMap map[typeName]methodName

type candidates struct {
	pkgs    []pkgName
	funcs   map[pkgName][]funcName
	methods map[pkgName][]receiverMap
	vars    map[pkgName][]varName
	consts  map[pkgName][]constName
	types   map[pkgName][]typeName
}

func GenerateCandidates(path string) (*candidates, error) {
	node, err := analyze(path)
	if err != nil {
		return nil, err
	}
	cs := convertFromNodeToCandidates(node)
	return &cs, nil
}
