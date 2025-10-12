package storage

import (
	"fmt"

	"github.com/microcosm-cc/bluemonday"
)

// ValidatePromptNotes validates and sanitizes human prompt notes
func ValidatePromptNotes(notes string) (string, error) {
	if len(notes) > 5000 {
		return "", fmt.Errorf("prompt notes exceed maximum length of 5000 characters")
	}

	p := bluemonday.UGCPolicy()
	return p.Sanitize(notes), nil
}
