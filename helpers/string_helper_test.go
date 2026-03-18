package helpers

import (
	"strings"
	"testing"
)

func TestSliceContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{"present", []string{"a", "b", "c"}, "b", true},
		{"absent", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SliceContainsString(tt.slice, tt.s); got != tt.expected {
				t.Errorf("SliceContainsString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRemoveStringFromSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		s        string
		expected []string
	}{
		{"removes existing", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{"no-op for absent", []string{"a", "b", "c"}, "d", []string{"a", "b", "c"}},
		{"empty slice", []string{}, "a", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveStringFromSlice(tt.slice, tt.s)
			if len(got) != len(tt.expected) {
				t.Errorf("RemoveStringFromSlice() = %v, want %v", got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("RemoveStringFromSlice()[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestSanitiseDbValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"hyphens to underscores", "my-db-name", "my_db_name"},
		{"removes special chars", "my@db!name#", "mydbname"},
		{"empty string", "", ""},
		{"already clean", "my_db_name", "my_db_name"},
		{"mixed", "my-db@name!123", "my_dbname123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitiseDbValue(tt.value); got != tt.expected {
				t.Errorf("SanitiseDbValue(%q) = %q, want %q", tt.value, got, tt.expected)
			}
		})
	}
}

func TestSanitiseAndShortenDbValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		maxLength int
		wantShort bool
	}{
		{"under max length", "short", 10, false},
		{"at max length", "exactly10!", 10, false},
		{"over max length - triggers hash", "this-is-a-very-long-database-name-that-exceeds-the-limit", 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitiseAndShortenDbValue(tt.value, tt.maxLength)
			if len(got) > tt.maxLength {
				t.Errorf("SanitiseAndShortenDbValue() length = %d, want <= %d", len(got), tt.maxLength)
			}
			if tt.wantShort && len(got) != 10 {
				t.Errorf("SanitiseAndShortenDbValue() hashed length = %d, want 10", len(got))
			}
		})
	}
}

func TestShortenHumanReadableValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		maxLength int
	}{
		{"under max length", "short", 20},
		{"over max length", "this-is-a-very-long-value-that-exceeds-the-maximum-length-allowed", 30},
		{"maxLength smaller than hash length falls back to hash only", "this-is-a-very-long-value", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortenHumanReadableValue(tt.value, tt.maxLength)
			switch tt.name {
			case "under max length":
				if got != tt.value {
					t.Errorf("ShortenHumanReadableValue() = %q, want %q (unchanged)", got, tt.value)
				}
			case "over max length":
				if len(got) > tt.maxLength {
					t.Errorf("ShortenHumanReadableValue() length = %d, want <= %d", len(got), tt.maxLength)
				}
				if !strings.Contains(got, "-") {
					t.Errorf("ShortenHumanReadableValue() = %q, expected prefix-dash-hash format", got)
				}
			case "maxLength smaller than hash length falls back to hash only":
				if len(got) != 10 {
					t.Errorf("ShortenHumanReadableValue() length = %d, want 10 (hash-only fallback)", len(got))
				}
			}
		})
	}
}

func TestMakeObjectName(t *testing.T) {
	tests := []struct {
		name     string
		baseName string
		suffixes []string
	}{
		{"no suffixes", "mysite", nil},
		{"multiple suffixes", "mysite", []string{"svc", "web"}},
		{"over 63 chars triggers shortening", strings.Repeat("a", 60), []string{"suffix"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeObjectName(tt.baseName, tt.suffixes...)
			if len(got) > 63 {
				t.Errorf("MakeObjectName() length = %d, want <= 63", len(got))
			}
			if tt.name == "no suffixes" && got != tt.baseName {
				t.Errorf("MakeObjectName() = %q, want %q", got, tt.baseName)
			}
			if tt.name == "multiple suffixes" {
				if !strings.HasSuffix(got, "-svc-web") {
					t.Errorf("MakeObjectName() = %q, want suffix -svc-web", got)
				}
			}
		})
	}
}

func TestGetKeysFromStringBoolMap(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]bool
		wantLen int
	}{
		{"returns keys", map[string]bool{"a": true, "b": false}, 2},
		{"empty map", map[string]bool{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetKeysFromStringBoolMap(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("GetKeysFromStringBoolMap() length = %d, want %d", len(got), tt.wantLen)
			}
			for _, s := range got {
				if _, ok := tt.input[s]; !ok {
					t.Errorf("GetKeysFromStringBoolMap() returned unexpected key %q", s)
				}
			}
		})
	}
}
