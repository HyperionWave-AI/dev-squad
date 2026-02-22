package serialization

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaDeserializer_NonMapSchemas(t *testing.T) {
	type typedSchema struct {
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties"`
	}

	sd := NewSchemaDeserializer(typedSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}, false)
	require.NotNil(t, sd.schema)
	assert.Equal(t, "object", sd.schema["type"])

	type unmarshalableSchema struct {
		Ch chan int `json:"ch"`
	}
	sd = NewSchemaDeserializer(unmarshalableSchema{Ch: make(chan int)}, true)
	assert.Nil(t, sd.schema)
	assert.True(t, sd.strictMode)
}

func TestSchemaDeserializer_DeserializeEdgeCases(t *testing.T) {
	sd := NewSchemaDeserializer(nil, false)

	for _, input := range []string{"", "null", "{}"} {
		result, err := sd.Deserialize([]byte(input))
		require.NoError(t, err)
		assert.Empty(t, result)
	}

	raw, err := sd.Deserialize([]byte(`{"key":"value","count":2}`))
	require.NoError(t, err)
	assert.Equal(t, "value", raw["key"])
	assert.Equal(t, float64(2), raw["count"])
}

func TestSchemaDeserializer_DeserializeDoubleEncodedAndErrors(t *testing.T) {
	sd := NewSchemaDeserializer(nil, false)

	result, err := sd.Deserialize([]byte(`"{\"x\":1}"`))
	require.NoError(t, err)
	assert.Equal(t, float64(1), result["x"])

	_, err = sd.Deserialize([]byte(`"{broken-json}"`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal arguments")

	_, err = sd.Deserialize([]byte(`not-json`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal arguments")
}

func TestSchemaDeserializer_TransformValueFallbackBranches(t *testing.T) {
	sd := NewSchemaDeserializer(nil, false)

	got, err := sd.transformValue(7, "not-a-map")
	require.NoError(t, err)
	assert.Equal(t, 7, got)

	got, err = sd.transformValue("x", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, "x", got)

	got, err = sd.transformValue("x", map[string]interface{}{"type": "unknown"})
	require.NoError(t, err)
	assert.Equal(t, "x", got)
}

func TestSchemaDeserializer_CoercionEdgeBranches(t *testing.T) {
	sd := NewSchemaDeserializer(nil, false)

	text, err := sd.coerceToString(float64(42))
	require.NoError(t, err)
	assert.Equal(t, "42", text)

	text, err = sd.coerceToString(nil)
	require.NoError(t, err)
	assert.Equal(t, "", text)

	text, err = sd.coerceToString(map[string]interface{}{"k": "v"})
	require.NoError(t, err)
	assert.Contains(t, text, `"k":"v"`)

	text, err = sd.coerceToString(make(chan int))
	require.NoError(t, err)
	assert.NotEmpty(t, text)

	number, err := sd.coerceToNumber(3, false)
	require.NoError(t, err)
	assert.Equal(t, 3.0, number)

	number, err = sd.coerceToNumber("7.25", true)
	require.NoError(t, err)
	assert.Equal(t, 7, number)

	_, err = sd.coerceToNumber("not-a-number", false)
	require.Error(t, err)

	_, err = sd.coerceToNumber(struct{}{}, false)
	require.Error(t, err)

	boolean, err := sd.coerceToBoolean("no")
	require.NoError(t, err)
	assert.False(t, boolean)

	boolean, err = sd.coerceToBoolean(nil)
	require.NoError(t, err)
	assert.False(t, boolean)

	_, err = sd.coerceToBoolean("maybe")
	require.Error(t, err)

	_, err = sd.coerceToBoolean([]int{1})
	require.Error(t, err)
}

func TestSchemaDeserializer_ArrayAndObjectErrorBranches(t *testing.T) {
	strict := NewSchemaDeserializer(nil, true)
	nonStrict := NewSchemaDeserializer(nil, false)

	itemSchema := map[string]interface{}{
		"items": map[string]interface{}{"type": "integer"},
	}

	_, err := strict.coerceToArray([]interface{}{"bad"}, itemSchema)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "array item 0")

	arr, err := nonStrict.coerceToArray([]interface{}{"bad"}, itemSchema)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{nil}, arr)

	arr, err = nonStrict.coerceToArray("not-json-array", itemSchema)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"not-json-array"}, arr)

	arr, err = nonStrict.coerceToArray(nil, itemSchema)
	require.NoError(t, err)
	assert.Empty(t, arr)

	arr, err = nonStrict.coerceToArray(9, itemSchema)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{9}, arr)

	objectSchema := map[string]interface{}{
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "integer"},
		},
	}
	obj, err := nonStrict.coerceToObject(map[string]interface{}{"count": "12"}, objectSchema)
	require.NoError(t, err)
	assert.Equal(t, 12, obj["count"])

	obj, err = nonStrict.coerceToObject(nil, objectSchema)
	require.NoError(t, err)
	assert.Empty(t, obj)

	_, err = nonStrict.coerceToObject(`{broken-json`, objectSchema)
	require.Error(t, err)

	_, err = nonStrict.coerceToObject(3.14, objectSchema)
	require.Error(t, err)
}

func TestSchemaDeserializer_HelperMethodsBranches(t *testing.T) {
	sd := NewSchemaDeserializer(nil, false)

	props, ok := sd.getSchemaProperties()
	assert.False(t, ok)
	assert.Nil(t, props)
	assert.Nil(t, sd.getRequiredFields())
	assert.True(t, sd.allowsAdditionalProperties())
	assert.Nil(t, sd.getDefaultValue("not-a-map"))

	withDefaults := NewSchemaDeserializer(map[string]interface{}{
		"properties":           "not-a-map",
		"required":             "not-a-slice",
		"additionalProperties": "not-a-bool",
	}, false)

	props, ok = withDefaults.getSchemaProperties()
	assert.False(t, ok)
	assert.Nil(t, props)
	assert.Nil(t, withDefaults.getRequiredFields())
	assert.True(t, withDefaults.allowsAdditionalProperties())
	assert.Equal(t, "v", withDefaults.getDefaultValue(map[string]interface{}{"default": "v"}))
	assert.Nil(t, withDefaults.getDefaultValue(map[string]interface{}{"other": 1}))

	assert.True(t, contains([]string{"a", "b"}, "a"))
	assert.False(t, contains([]string{"a", "b"}, "z"))
}

func TestNormalizeToolArguments_ErrorPath(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "integer"},
		},
	}

	_, err := NormalizeToolArguments(json.RawMessage(`{invalid`), schema)
	require.Error(t, err)
}
