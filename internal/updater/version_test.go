package updater

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		major int
		minor int
		patch int
		ok    bool
	}{
		{"v1.2.3", 1, 2, 3, true},
		{"1.2.3", 1, 2, 3, true},
		{"v0.0.1", 0, 0, 1, true},
		{"v10.20.30", 10, 20, 30, true},
		{"dev", 0, 0, 0, false},
		{"v1", 0, 0, 0, false},
		{"v1.2", 0, 0, 0, false},
		{"", 0, 0, 0, false},
		{"abc", 0, 0, 0, false},
	}
	for _, tt := range tests {
		got, ok := ParseVersion(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseVersion(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok && got != (Version{tt.major, tt.minor, tt.patch}) {
			t.Errorf("ParseVersion(%q) = %+v, want {%d,%d,%d}", tt.input, got, tt.major, tt.minor, tt.patch)
		}
	}
}

func TestVersionGreaterThan(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v2.0.0", "v1.9.9", true},
		{"v1.3.0", "v1.2.9", true},
		{"v1.2.4", "v1.2.3", true},
		{"v1.2.3", "v1.2.3", false},
		{"v1.2.2", "v1.2.3", false},
		{"v0.9.0", "v1.0.0", false},
	}
	for _, tt := range tests {
		a, _ := ParseVersion(tt.a)
		b, _ := ParseVersion(tt.b)
		if got := a.GreaterThan(b); got != tt.want {
			t.Errorf("ParseVersion(%q).GreaterThan(ParseVersion(%q)) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
