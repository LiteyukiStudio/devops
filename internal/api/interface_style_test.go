package api

import "testing"

func TestNormalizeUserInterfaceStyle(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
		valid bool
	}{
		{input: "", want: "", valid: true},
		{input: " Minimal ", want: "minimal", valid: true},
		{input: "themed", want: "themed", valid: true},
		{input: "custom", want: "custom", valid: false},
	} {
		got, valid := normalizeUserInterfaceStyle(test.input)
		if got != test.want || valid != test.valid {
			t.Fatalf("normalizeUserInterfaceStyle(%q) = %q, %v; want %q, %v", test.input, got, valid, test.want, test.valid)
		}
	}
}
