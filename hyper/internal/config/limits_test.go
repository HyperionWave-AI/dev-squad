package config

import "testing"

func TestFormatSizeLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{name: "bytes", input: 999, want: "999B"},
		{name: "kilobytes", input: 2 * 1024, want: "2KB"},
		{name: "megabytes", input: 3 * 1024 * 1024, want: "3MB"},
		{name: "gigabytes", input: 4 * 1024 * 1024 * 1024, want: "4GB"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSizeLimit(tc.input)
			if got != tc.want {
				t.Fatalf("FormatSizeLimit(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  string
	}{
		{name: "bytes", input: 999, want: "999B"},
		{name: "kilobytes", input: 1536, want: "1.5KB"},
		{name: "megabytes", input: 1572864, want: "1.5MB"},
		{name: "gigabytes", input: 1610612736, want: "1.5GB"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSize(tc.input)
			if got != tc.want {
				t.Fatalf("FormatSize(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGetMaxContextSize(t *testing.T) {
	t.Run("default when unset", func(t *testing.T) {
		t.Setenv("MAX_CONTEXT_SIZE", "")
		if got := GetMaxContextSize(); got != DefaultMaxContextSize {
			t.Fatalf("GetMaxContextSize() = %d, want %d", got, DefaultMaxContextSize)
		}
	})

	t.Run("uses positive env value", func(t *testing.T) {
		t.Setenv("MAX_CONTEXT_SIZE", "123456")
		if got := GetMaxContextSize(); got != 123456 {
			t.Fatalf("GetMaxContextSize() = %d, want %d", got, 123456)
		}
	})

	t.Run("falls back on invalid env value", func(t *testing.T) {
		t.Setenv("MAX_CONTEXT_SIZE", "abc")
		if got := GetMaxContextSize(); got != DefaultMaxContextSize {
			t.Fatalf("GetMaxContextSize() = %d, want %d", got, DefaultMaxContextSize)
		}
	})

	t.Run("falls back on zero or negative env value", func(t *testing.T) {
		t.Setenv("MAX_CONTEXT_SIZE", "0")
		if got := GetMaxContextSize(); got != DefaultMaxContextSize {
			t.Fatalf("GetMaxContextSize() = %d, want %d", got, DefaultMaxContextSize)
		}

		t.Setenv("MAX_CONTEXT_SIZE", "-42")
		if got := GetMaxContextSize(); got != DefaultMaxContextSize {
			t.Fatalf("GetMaxContextSize() = %d, want %d", got, DefaultMaxContextSize)
		}
	})
}
