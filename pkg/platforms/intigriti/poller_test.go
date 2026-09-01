package intigriti

import (
	"sort"
	"testing"
)

func TestGetCategoryIDs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{"all", "all", nil},
		{"empty", "", nil},
		{"invalid falls back to all", "nonsense", nil},
		{"single", "url", []int{1}},
		{"comma separated", "url,cidr", []int{1, 4}},
		{"spaces and casing", " URL , Wildcard ", []int{1, 7}},
		{"duplicates", "url,url", []int{1}},
		{"invalid ones are ignored", "url,nonsense", []int{1}},
		{"mobile categories", "android,ios", []int{2, 3}},
		{"unsupported on intigriti", "blockchain", []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCategoryIDs(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("getCategoryIDs(%q) = %v, want nil (no filtering)", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("getCategoryIDs(%q) = nil (no filtering), want %v", tt.input, tt.want)
			}
			sort.Ints(got)
			if len(got) != len(tt.want) {
				t.Fatalf("getCategoryIDs(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("getCategoryIDs(%q) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}
