package storage

import (
	"strings"
	"testing"
)

func TestValidatePromptNotes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantOutput  string
		wantError   bool
		errorContains string
	}{
		{
			name:       "Valid markdown with headers",
			input:      "# Header\n## Subheader\nSome text",
			wantOutput: "# Header\n## Subheader\nSome text",
			wantError:  false,
		},
		{
			name:       "Valid markdown with code blocks",
			input:      "```go\nfunc main() {}\n```",
			wantOutput: "```go\nfunc main() {}\n```",
			wantError:  false,
		},
		{
			name:       "Valid markdown with lists",
			input:      "- Item 1\n- Item 2\n* Item 3",
			wantOutput: "- Item 1\n- Item 2\n* Item 3",
			wantError:  false,
		},
		{
			name:       "Valid markdown with bold and italic",
			input:      "**bold** and *italic* text",
			wantOutput: "**bold** and *italic* text",
			wantError:  false,
		},
		{
			name:       "Valid markdown with links",
			input:      "[Link text](https://example.com)",
			wantOutput: "[Link text](https://example.com)",
			wantError:  false,
		},
		{
			name:       "Max length exactly 5000 characters",
			input:      strings.Repeat("a", 5000),
			wantOutput: strings.Repeat("a", 5000),
			wantError:  false,
		},
		{
			name:          "Over max length 5001 characters",
			input:         strings.Repeat("a", 5001),
			wantError:     true,
			errorContains: "exceed maximum length",
		},
		{
			name:       "HTML script tags stripped",
			input:      "Safe text <script>alert('xss')</script> more text",
			wantOutput: "Safe text  more text",
			wantError:  false,
		},
		{
			name:       "HTML div/p tags preserved by UGCPolicy",
			input:      "<div>Content</div><p>Paragraph</p>",
			wantOutput: "<div>Content</div><p>Paragraph</p>",
			wantError:  false,
		},
		{
			name:       "Dangerous onclick attributes stripped",
			input:      "<a href='#' onclick='alert(1)'>Link</a>",
			wantOutput: "Link",
			wantError:  false,
		},
		{
			name:       "Empty string",
			input:      "",
			wantOutput: "",
			wantError:  false,
		},
		{
			name:       "Whitespace only",
			input:      "   \n\t  ",
			wantOutput: "   \n\t  ",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutput, gotErr := ValidatePromptNotes(tt.input)

			// Check error expectation
			if tt.wantError {
				if gotErr == nil {
					t.Errorf("ValidatePromptNotes() expected error but got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(gotErr.Error(), tt.errorContains) {
					t.Errorf("ValidatePromptNotes() error = %v, want error containing %q", gotErr, tt.errorContains)
				}
				return
			}

			// No error expected
			if gotErr != nil {
				t.Errorf("ValidatePromptNotes() unexpected error = %v", gotErr)
				return
			}

			// Check output
			if gotOutput != tt.wantOutput {
				t.Errorf("ValidatePromptNotes() output = %q, want %q", gotOutput, tt.wantOutput)
			}
		})
	}
}
