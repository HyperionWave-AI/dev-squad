package serialization

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// SchemaDeserializer handles tool argument deserialization based on JSON schema
type SchemaDeserializer struct {
	schema     map[string]interface{}
	strictMode bool // When true, enforces required fields and type constraints
}

// NewSchemaDeserializer creates a new schema-aware deserializer
func NewSchemaDeserializer(schema interface{}, strictMode bool) *SchemaDeserializer {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		// Try to convert if it's a different type
		if data, err := json.Marshal(schema); err == nil {
			json.Unmarshal(data, &schemaMap)
		}
	}

	return &SchemaDeserializer{
		schema:     schemaMap,
		strictMode: strictMode,
	}
}

// DeserializeToolArguments deserializes tool arguments according to the provided schema
func DeserializeToolArguments(arguments json.RawMessage, schema interface{}) (map[string]interface{}, error) {
	deserializer := NewSchemaDeserializer(schema, false)
	return deserializer.Deserialize(arguments)
}

// DeserializeToolArgumentsStrict deserializes with strict schema validation
func DeserializeToolArgumentsStrict(arguments json.RawMessage, schema interface{}) (map[string]interface{}, error) {
	deserializer := NewSchemaDeserializer(schema, true)
	return deserializer.Deserialize(arguments)
}

// Deserialize performs the actual deserialization based on schema
func (sd *SchemaDeserializer) Deserialize(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 || string(data) == "null" || string(data) == "{}" {
		// Return empty map for null/empty arguments
		return make(map[string]interface{}), nil
	}

	// First, try to unmarshal as-is
	var rawArgs map[string]interface{}
	if err := json.Unmarshal(data, &rawArgs); err != nil {
		// Try to extract from string if it's double-encoded
		var strData string
		if json.Unmarshal(data, &strData) == nil {
			if err := json.Unmarshal([]byte(strData), &rawArgs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
		}
	}

	// If no schema, return raw arguments
	if sd.schema == nil {
		return rawArgs, nil
	}

	// Apply schema-based transformations
	result, err := sd.applySchema(rawArgs)
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return result, nil
}

// applySchema applies schema rules to the raw arguments
func (sd *SchemaDeserializer) applySchema(rawArgs map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get properties from schema
	properties, hasProperties := sd.getSchemaProperties()
	required := sd.getRequiredFields()

	// If schema has no properties defined, return raw args
	if !hasProperties {
		return rawArgs, nil
	}

	// Process each property according to schema
	for propName, propSchema := range properties {
		value, exists := rawArgs[propName]

		// Handle required fields
		if sd.strictMode && contains(required, propName) && !exists {
			return nil, fmt.Errorf("required field '%s' is missing", propName)
		}

		if exists {
			// Transform value based on schema type
			transformedValue, err := sd.transformValue(value, propSchema)
			if err != nil && sd.strictMode {
				return nil, fmt.Errorf("field '%s': %w", propName, err)
			}
			result[propName] = transformedValue
		} else if propSchema != nil {
			// Check for default value in schema
			if defaultValue := sd.getDefaultValue(propSchema); defaultValue != nil {
				result[propName] = defaultValue
			}
		}
	}

	// Handle additional properties
	additionalAllowed := sd.allowsAdditionalProperties()
	for key, value := range rawArgs {
		if _, defined := properties[key]; !defined {
			if sd.strictMode && !additionalAllowed {
				return nil, fmt.Errorf("additional property '%s' not allowed by schema", key)
			}
			if additionalAllowed {
				result[key] = value
			}
		}
	}

	return result, nil
}

// transformValue transforms a value according to its schema definition
func (sd *SchemaDeserializer) transformValue(value interface{}, propSchema interface{}) (interface{}, error) {
	schemaMap, ok := propSchema.(map[string]interface{})
	if !ok {
		// No schema details, return as-is
		return value, nil
	}

	schemaType, hasType := schemaMap["type"].(string)
	if !hasType {
		// No type specified, return as-is
		return value, nil
	}

	switch schemaType {
	case "string":
		return sd.coerceToString(value)
	case "number", "integer":
		return sd.coerceToNumber(value, schemaType == "integer")
	case "boolean":
		return sd.coerceToBoolean(value)
	case "array":
		return sd.coerceToArray(value, schemaMap)
	case "object":
		return sd.coerceToObject(value, schemaMap)
	default:
		return value, nil
	}
}

// coerceToString converts value to string
func (sd *SchemaDeserializer) coerceToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10), nil
		}
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case nil:
		return "", nil
	default:
		// Try JSON encoding for complex types
		if data, err := json.Marshal(v); err == nil {
			return string(data), nil
		}
		return fmt.Sprintf("%v", v), nil
	}
}

// coerceToNumber converts value to number
func (sd *SchemaDeserializer) coerceToNumber(value interface{}, asInteger bool) (interface{}, error) {
	switch v := value.(type) {
	case float64:
		if asInteger {
			return int(v), nil
		}
		return v, nil
	case int:
		if asInteger {
			return v, nil
		}
		return float64(v), nil
	case string:
		// Try to parse string as number
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			if asInteger {
				return int(f), nil
			}
			return f, nil
		}
		return nil, fmt.Errorf("cannot convert string '%s' to number", v)
	case bool:
		if v {
			if asInteger {
				return 1, nil
			}
			return 1.0, nil
		}
		if asInteger {
			return 0, nil
		}
		return 0.0, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to number", value)
	}
}

// coerceToBoolean converts value to boolean
func (sd *SchemaDeserializer) coerceToBoolean(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		lower := strings.ToLower(v)
		if lower == "true" || lower == "yes" || lower == "1" {
			return true, nil
		}
		if lower == "false" || lower == "no" || lower == "0" || lower == "" {
			return false, nil
		}
		return false, fmt.Errorf("cannot convert string '%s' to boolean", v)
	case float64:
		return v != 0, nil
	case int:
		return v != 0, nil
	case nil:
		return false, nil
	default:
		return false, fmt.Errorf("cannot convert %T to boolean", value)
	}
}

// coerceToArray converts value to array
func (sd *SchemaDeserializer) coerceToArray(value interface{}, schemaMap map[string]interface{}) ([]interface{}, error) {
	switch v := value.(type) {
	case []interface{}:
		// Process array items if item schema is defined
		if itemSchema, hasItems := schemaMap["items"]; hasItems {
			result := make([]interface{}, len(v))
			for i, item := range v {
				transformed, err := sd.transformValue(item, itemSchema)
				if err != nil && sd.strictMode {
					return nil, fmt.Errorf("array item %d: %w", i, err)
				}
				result[i] = transformed
			}
			return result, nil
		}
		return v, nil
	case string:
		// Try to parse JSON array from string
		var arr []interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			return sd.coerceToArray(arr, schemaMap)
		}
		// Single string becomes array with one element
		return []interface{}{v}, nil
	case nil:
		return []interface{}{}, nil
	default:
		// Single value becomes array with one element
		return []interface{}{value}, nil
	}
}

// coerceToObject converts value to object
func (sd *SchemaDeserializer) coerceToObject(value interface{}, schemaMap map[string]interface{}) (map[string]interface{}, error) {
	switch v := value.(type) {
	case map[string]interface{}:
		// If nested properties are defined, apply them
		if props, hasProps := schemaMap["properties"].(map[string]interface{}); hasProps {
			deserializer := &SchemaDeserializer{
				schema: map[string]interface{}{
					"type":       "object",
					"properties": props,
				},
				strictMode: sd.strictMode,
			}
			return deserializer.applySchema(v)
		}
		return v, nil
	case string:
		// Try to parse JSON object from string
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(v), &obj); err == nil {
			return sd.coerceToObject(obj, schemaMap)
		}
		return nil, fmt.Errorf("cannot parse string as object")
	case nil:
		return make(map[string]interface{}), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to object", value)
	}
}

// Helper methods

func (sd *SchemaDeserializer) getSchemaProperties() (map[string]interface{}, bool) {
	if sd.schema == nil {
		return nil, false
	}

	props, ok := sd.schema["properties"].(map[string]interface{})
	return props, ok
}

func (sd *SchemaDeserializer) getRequiredFields() []string {
	if sd.schema == nil {
		return nil
	}

	required, ok := sd.schema["required"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(required))
	for _, r := range required {
		if str, ok := r.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func (sd *SchemaDeserializer) allowsAdditionalProperties() bool {
	if sd.schema == nil {
		return true
	}

	additional, exists := sd.schema["additionalProperties"]
	if !exists {
		return true // Default to allowing additional properties
	}

	if boolVal, ok := additional.(bool); ok {
		return boolVal
	}

	return true
}

func (sd *SchemaDeserializer) getDefaultValue(propSchema interface{}) interface{} {
	schemaMap, ok := propSchema.(map[string]interface{})
	if !ok {
		return nil
	}

	defaultVal, exists := schemaMap["default"]
	if exists {
		return defaultVal
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidateAgainstSchema validates that arguments conform to the schema without transformation
func ValidateAgainstSchema(arguments json.RawMessage, schema interface{}) error {
	deserializer := NewSchemaDeserializer(schema, true)
	_, err := deserializer.Deserialize(arguments)
	return err
}

// NormalizeToolArguments normalizes tool arguments to match schema expectations
// This is useful for handling AI-generated arguments that might not perfectly match the schema
func NormalizeToolArguments(arguments json.RawMessage, schema interface{}) (json.RawMessage, error) {
	result, err := DeserializeToolArguments(arguments, schema)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}
