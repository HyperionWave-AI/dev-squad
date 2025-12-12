package validation

import (
	"context"
)

// ValidationPlugin wraps the validator with enable/disable functionality
// When disabled, all validation operations are no-ops
type ValidationPlugin struct {
	validator *CodeValidator
	enabled   bool
}

// NewValidationPlugin creates a new validation plugin (default: disabled)
func NewValidationPlugin(validator *CodeValidator) *ValidationPlugin {
	return &ValidationPlugin{
		validator: validator,
		enabled:   false, // Default OFF for backward compatibility
	}
}

// SetEnabled enables or disables the validation plugin
func (p *ValidationPlugin) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// IsEnabled returns whether the plugin is currently enabled
func (p *ValidationPlugin) IsEnabled() bool {
	return p.enabled
}

// ValidateIfEnabled runs validation only if plugin is enabled
// If disabled, returns success immediately without running any checks
func (p *ValidationPlugin) ValidateIfEnabled(ctx context.Context, files []string) (*ValidationResult, error) {
	if !p.enabled {
		// Plugin disabled - return success without running validation
		// This makes the code behave as if validation doesn't exist
		return &ValidationResult{
			Passed:  true,
			Skipped: true,
			Message: "Validation skipped - Error Prevention Mode is OFF",
		}, nil
	}

	// Plugin enabled - run actual validation
	return p.validator.ValidateFiles(ctx, files)
}

// MustBeEnabled panics if plugin is not enabled
// Use this in tests to ensure validation is properly configured
func (p *ValidationPlugin) MustBeEnabled() {
	if !p.enabled {
		panic("ValidationPlugin must be enabled for this operation")
	}
}
