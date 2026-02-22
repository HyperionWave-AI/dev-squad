package validation

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestValidationPlugin_DefaultAndEnableToggle(t *testing.T) {
	validator := NewCodeValidator(zap.NewNop(), t.TempDir())
	plugin := NewValidationPlugin(validator)

	if plugin.IsEnabled() {
		t.Fatal("expected plugin to be disabled by default")
	}

	plugin.SetEnabled(true)
	if !plugin.IsEnabled() {
		t.Fatal("expected plugin to be enabled after SetEnabled(true)")
	}
}

func TestValidationPlugin_ValidateIfEnabled_Disabled(t *testing.T) {
	plugin := NewValidationPlugin(NewCodeValidator(zap.NewNop(), t.TempDir()))

	result, err := plugin.ValidateIfEnabled(context.Background(), []string{"a.go"})
	if err != nil {
		t.Fatalf("ValidateIfEnabled returned error: %v", err)
	}
	if !result.Passed || !result.Skipped {
		t.Fatalf("expected skipped success result, got %+v", result)
	}
}

func TestValidationPlugin_ValidateIfEnabled_Enabled(t *testing.T) {
	plugin := NewValidationPlugin(NewCodeValidator(zap.NewNop(), t.TempDir()))
	plugin.SetEnabled(true)

	result, err := plugin.ValidateIfEnabled(context.Background(), nil)
	if err != nil {
		t.Fatalf("ValidateIfEnabled returned error: %v", err)
	}
	if !result.Passed {
		t.Fatalf("expected passed result, got %+v", result)
	}
}

func TestValidationPlugin_MustBeEnabled(t *testing.T) {
	plugin := NewValidationPlugin(NewCodeValidator(zap.NewNop(), t.TempDir()))

	defer func() {
		if recover() == nil {
			t.Fatal("expected MustBeEnabled to panic when disabled")
		}
	}()
	plugin.MustBeEnabled()
}

func TestValidationPlugin_MustBeEnabled_NoPanicWhenEnabled(t *testing.T) {
	plugin := NewValidationPlugin(NewCodeValidator(zap.NewNop(), t.TempDir()))
	plugin.SetEnabled(true)

	plugin.MustBeEnabled()
}
