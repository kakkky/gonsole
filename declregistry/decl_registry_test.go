package declregistry

import (
	"testing"

	gotypes "go/types"

	"github.com/google/go-cmp/cmp"
	"github.com/kakkky/gonsole/types"
)

func TestDeclRegistry_Register(t *testing.T) {
	tests := []struct {
		name          string
		declName      types.DeclName
		declType      func() gotypes.Type // gotypes.Typeを返す関数
		existingDecls []Decl
		expected      []Decl
	}{
		{
			name:     "int variable",
			declName: "a",
			declType: func() gotypes.Type {
				return gotypes.NewNamed(gotypes.NewTypeName(0, nil, "int", nil), nil, nil)
			},
			expected: []Decl{{Name: "a", TypeName: "int"}},
		},
		{
			name:     "struct variable",
			declName: "s",
			declType: func() gotypes.Type {
				return gotypes.NewNamed(gotypes.NewTypeName(0, nil, "Struct", nil), nil, nil)
			},
			expected: []Decl{{Name: "s", TypeName: "Struct"}},
		},
		{
			name:     "pointer to struct",
			declName: "p",
			declType: func() gotypes.Type {
				named := gotypes.NewNamed(gotypes.NewTypeName(0, nil, "Struct", nil), nil, nil)
				return gotypes.NewPointer(named)
			},
			expected: []Decl{{Name: "p", TypeName: "Struct", Pointered: true}},
		},
		{
			name:     "string variable",
			declName: "b",
			declType: func() gotypes.Type {
				return gotypes.NewNamed(gotypes.NewTypeName(0, nil, "string", nil), nil, nil)
			},
			expected: []Decl{{Name: "b", TypeName: "string"}},
		},
		{
			name:     "method assignment",
			declName: "b",
			declType: func() gotypes.Type {
				return gotypes.NewNamed(gotypes.NewTypeName(0, nil, "Struct", nil), nil, nil)
			},
			existingDecls: []Decl{{Name: "a", TypeName: "Struct"}},
			expected:      []Decl{{Name: "a", TypeName: "Struct"}, {Name: "b", TypeName: "Struct"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sut := NewRegistry()
			if tt.existingDecls != nil {
				sut.Decls = tt.existingDecls
			}
			err := sut.Register(tt.declName, tt.declType())
			if err != nil {
				t.Fatalf("Register() returned an error: %v", err)
			}
			if diff := cmp.Diff(tt.expected, sut.Decls); diff != "" {
				t.Errorf("Register() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRegistry_IsRegisteredDecl(t *testing.T) {
	tests := []struct {
		name          string
		existingDecls []Decl
		checkName     types.DeclName
		expectedFound bool
	}{
		{
			name: "found registered declaration",
			existingDecls: []Decl{
				{
					Name:        "var1",
					TypeName:    "int",
					TypePkgName: "",
				},
				{
					Name:        "struct1",
					TypeName:    "Struct",
					TypePkgName: "declregistry",
				},
			},
			checkName:     "var1",
			expectedFound: true,
		},
		{
			name: "found registered declaration (from multiple declarations)",
			existingDecls: []Decl{
				{
					Name:        "var1",
					TypeName:    "int",
					TypePkgName: "",
				},
				{
					Name:        "struct1",
					TypeName:    "Struct",
					TypePkgName: "declregistry",
				},
			},
			checkName:     "struct1",
			expectedFound: true,
		},
		{
			name: "not found registered declaration",
			existingDecls: []Decl{
				{
					Name:        "var1",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
			checkName:     "unknown",
			expectedFound: false,
		},
		{
			name:          "not found registered declaration (empty list)",
			existingDecls: []Decl{},
			checkName:     "var1",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Registryの準備
			dr := &DeclRegistry{
				Decls: tt.existingDecls,
			}

			// テスト対象メソッドの実行
			result := dr.IsRegisteredDecl(tt.checkName)

			// 結果の検証
			if result != tt.expectedFound {
				t.Errorf("IsRegisteredDecl(%s) = %v, want %v",
					tt.checkName, result, tt.expectedFound)
			}
		})
	}
}
