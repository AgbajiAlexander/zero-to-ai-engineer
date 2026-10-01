package learner

import "testing"

func TestNormalizeDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trims surrounding whitespace",
			input: "  Alice Engineer  ",
			want:  "Alice Engineer",
		},
		{
			name:  "collapses repeated spaces",
			input: "Alice    Engineer",
			want:  "Alice Engineer",
		},
		{
			name:  "preserves Unicode names",
			input: "Chinwe Okafor",
			want:  "Chinwe Okafor",
		},
		{
			name:  "preserves non-Latin characters",
			input: "山田 太郎",
			want:  "山田 太郎",
		},
		{
			name:  "normalises Unicode whitespace",
			input: "\u00A0Alice\u00A0Engineer\u00A0",
			want:  "Alice Engineer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDisplayName(tt.input)
			if err != nil {
				t.Fatalf("NormalizeDisplayName() returned unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("NormalizeDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeDisplayNameRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "whitespace only",
			input: "   \t ",
		},
		{
			name:  "newline control character",
			input: "Alice\nEngineer",
		},
		{
			name:  "tab control character",
			input: "Alice\tEngineer",
		},
		{
			name:  "81 Unicode code points",
			input: repeatedCharacter("a", 81),
		},
		{
			name:  "invalid UTF-8",
			input: string([]byte{0xff, 0xfe}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDisplayName(tt.input)
			if err == nil {
				t.Fatalf("NormalizeDisplayName(%q) = %q, want an error", tt.input, got)
			}

			if got != "" {
				t.Errorf("NormalizeDisplayName() returned %q on error, want empty string", got)
			}
		})
	}
}

func TestNormalizeDisplayNameAcceptsMaximumLength(t *testing.T) {
	input := repeatedCharacter("a", 80)

	got, err := NormalizeDisplayName(input)
	if err != nil {
		t.Fatalf("NormalizeDisplayName() rejected 80 code points: %v", err)
	}

	if got != input {
		t.Errorf("NormalizeDisplayName() = %q, want %q", got, input)
	}
}

func repeatedCharacter(value string, count int) string {
	result := ""

	for i := 0; i < count; i++ {
		result += value
	}

	return result
}
