package completer

import (
	"testing"

	"github.com/kakkky/go-prompt"
)

func TestSuggestionBuilder_build(t *testing.T) {
	tests := []struct {
		name        string
		rawInput    string
		candidate   string
		suggestType suggestType
		description string
		appendText  []string
		want        prompt.Suggest
	}{
		// Package suggestions
		{
			name:        "Package: 完全一致",
			rawInput:    "fmt",
			candidate:   "fmt",
			suggestType: suggestTypePackage,
			description: "fmt",
			want: prompt.Suggest{
				Text:        "fmt",
				DisplayText: "fmt",
				Description: "Package: fmt",
			},
		},
		{
			name:        "Package: 部分一致",
			rawInput:    "f",
			candidate:   "fmt",
			suggestType: suggestTypePackage,
			description: "fmt",
			want: prompt.Suggest{
				Text:        "fmt",
				DisplayText: "fmt",
				Description: "Package: fmt",
			},
		},
		{
			name:        "Package: 空文字列から",
			rawInput:    "",
			candidate:   "fmt",
			suggestType: suggestTypePackage,
			description: "fmt",
			want: prompt.Suggest{
				Text:        "fmt",
				DisplayText: "fmt",
				Description: "Package: fmt",
			},
		},

		// Function suggestions
		{
			name:        "Function: ドット後の部分一致",
			rawInput:    "fmt.Pr",
			candidate:   "Printf",
			suggestType: suggestTypeFunction,
			description: "Printf",
			want: prompt.Suggest{
				Text:        "fmt.Printf",
				DisplayText: "Printf",
				Description: "Function: Printf",
			},
		},
		{
			name:        "Function: ドット直後",
			rawInput:    "fmt.",
			candidate:   "Println",
			suggestType: suggestTypeFunction,
			description: "Println",
			want: prompt.Suggest{
				Text:        "fmt.Println",
				DisplayText: "Println",
				Description: "Function: Println",
			},
		},
		{
			name:        "Function: 完全一致",
			rawInput:    "fmt.Printf",
			candidate:   "Printf",
			suggestType: suggestTypeFunction,
			description: "Printf",
			want: prompt.Suggest{
				Text:        "fmt.Printf",
				DisplayText: "Printf",
				Description: "Function: Printf",
			},
		},
		{
			name:        "Function: 1文字一致",
			rawInput:    "fmt.P",
			candidate:   "Printf",
			suggestType: suggestTypeFunction,
			description: "Printf",
			want: prompt.Suggest{
				Text:        "fmt.Printf",
				DisplayText: "Printf",
				Description: "Function: Printf",
			},
		},

		// Struct suggestions with & prefix
		{
			name:        "Struct: & prefix付き部分一致",
			rawInput:    "&fmt.St",
			candidate:   "Stringer",
			suggestType: suggestTypeStruct,
			description: "Stringer",
			want: prompt.Suggest{
				Text:        "&fmt.Stringer",
				DisplayText: "Stringer",
				Description: "Struct: Stringer",
			},
		},
		{
			name:        "Struct: & prefix付きドット直後",
			rawInput:    "&os.",
			candidate:   "File",
			suggestType: suggestTypeStruct,
			description: "File",
			want: prompt.Suggest{
				Text:        "&os.File",
				DisplayText: "File",
				Description: "Struct: File",
			},
		},
		{
			name:        "Struct: & なし",
			rawInput:    "os.F",
			candidate:   "File",
			suggestType: suggestTypeStruct,
			description: "File",
			want: prompt.Suggest{
				Text:        "os.File",
				DisplayText: "File",
				Description: "Struct: File",
			},
		},

		// Variable suggestions
		{
			name:        "Variable: ドット後の部分一致",
			rawInput:    "pkg.va",
			candidate:   "variable",
			suggestType: suggestTypeVariable,
			description: "variable",
			want: prompt.Suggest{
				Text:        "pkg.variable",
				DisplayText: "variable",
				Description: "Variable: variable",
			},
		},
		{
			name:        "Variable: 変数宣言後",
			rawInput:    "x = pkg.",
			candidate:   "variable",
			suggestType: suggestTypeVariable,
			description: "variable",
			want: prompt.Suggest{
				Text:        "pkg.variable",
				DisplayText: "variable",
				Description: "Variable: variable",
			},
		},

		// Method suggestions
		{
			name:        "Method: レシーバ付き部分一致",
			rawInput:    "logger.In",
			candidate:   "Info",
			suggestType: suggestTypeMethod,
			description: "Info",
			want: prompt.Suggest{
				Text:        "logger.Info",
				DisplayText: "Info",
				Description: "Method: Info",
			},
		},
		{
			name:        "Method: ドット直後",
			rawInput:    "logger.",
			candidate:   "Info",
			suggestType: suggestTypeMethod,
			description: "Info",
			want: prompt.Suggest{
				Text:        "logger.Info",
				DisplayText: "Info",
				Description: "Method: Info",
			},
		},

		// Constant suggestions
		{
			name:        "Constant: ドット後の部分一致",
			rawInput:    "os.O_",
			candidate:   "O_RDONLY",
			suggestType: suggestTypeConstant,
			description: "O_RDONLY",
			want: prompt.Suggest{
				Text:        "os.O_RDONLY",
				DisplayText: "O_RDONLY",
				Description: "Constant: O_RDONLY",
			},
		},
		{
			name:        "Constant: ドット直後",
			rawInput:    "math.",
			candidate:   "Pi",
			suggestType: suggestTypeConstant,
			description: "Pi",
			want: prompt.Suggest{
				Text:        "math.Pi",
				DisplayText: "Pi",
				Description: "Constant: Pi",
			},
		},

		// Edge cases
		{
			name:        "Unknown type",
			rawInput:    "test",
			candidate:   "test",
			suggestType: suggestTypeUnknown,
			description: "test",
			want: prompt.Suggest{
				Text:        "test",
				DisplayText: "test",
				Description: "Unknown: test",
			},
		},
		{
			name:        "長い部分一致",
			rawInput:    "veryLongPackageNa",
			candidate:   "veryLongPackageName",
			suggestType: suggestTypePackage,
			description: "veryLongPackageName",
			want: prompt.Suggest{
				Text:        "veryLongPackageName",
				DisplayText: "veryLongPackageName",
				Description: "Package: veryLongPackageName",
			},
		},
		{
			name:        "複数ドット",
			rawInput:    "pkg.sub.F",
			candidate:   "Func",
			suggestType: suggestTypeFunction,
			description: "Func",
			want: prompt.Suggest{
				Text:        "pkg.sub.Func",
				DisplayText: "Func",
				Description: "Function: Func",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := newSuggestionBuilder(tt.rawInput)
			got := sb.build(tt.candidate, tt.suggestType, tt.description, tt.appendText...)

			if got.Text != tt.want.Text {
				t.Errorf("Text = %q, want %q", got.Text, tt.want.Text)
			}
			if got.DisplayText != tt.want.DisplayText {
				t.Errorf("DisplayText = %q, want %q", got.DisplayText, tt.want.DisplayText)
			}
			if got.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", got.Description, tt.want.Description)
			}
		})
	}
}
