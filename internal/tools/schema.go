package tools

import (
	"encoding/json"
)

// SchemaBuilder generates JSON Schema for tool parameters
type SchemaBuilder struct {
	schema map[string]any
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// AddString adds a string property
func (b *SchemaBuilder) AddString(name, description string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "string",
		"description": description,
	}
	if required {
		b.addRequired(name)
	}
	return b
}

// AddStringEnum adds a string property with enum values
func (b *SchemaBuilder) AddStringEnum(name, description string, values []string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
	if required {
		b.addRequired(name)
	}
	return b
}

// AddInteger adds an integer property
func (b *SchemaBuilder) AddInteger(name, description string, required bool, min, max *int) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	prop := map[string]any{
		"type":        "integer",
		"description": description,
	}
	if min != nil {
		prop["minimum"] = *min
	}
	if max != nil {
		prop["maximum"] = *max
	}
	props[name] = prop
	if required {
		b.addRequired(name)
	}
	return b
}

// AddNumber adds a number property
func (b *SchemaBuilder) AddNumber(name, description string, required bool, min, max *float64) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	prop := map[string]any{
		"type":        "number",
		"description": description,
	}
	if min != nil {
		prop["minimum"] = *min
	}
	if max != nil {
		prop["maximum"] = *max
	}
	props[name] = prop
	if required {
		b.addRequired(name)
	}
	return b
}

// AddBoolean adds a boolean property
func (b *SchemaBuilder) AddBoolean(name, description string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "boolean",
		"description": description,
	}
	if required {
		b.addRequired(name)
	}
	return b
}

// AddStringArray adds a string array property
func (b *SchemaBuilder) AddStringArray(name, description string, required bool) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "array",
		"description": description,
		"items": map[string]any{
			"type": "string",
		},
	}
	if required {
		b.addRequired(name)
	}
	return b
}

// AddObject adds an object property
func (b *SchemaBuilder) AddObject(name, description string, required bool, properties map[string]any) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	props[name] = map[string]any{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
	if required {
		b.addRequired(name)
	}
	return b
}

// AddObjectArray adds an array of objects
func (b *SchemaBuilder) AddObjectArray(name, description string, required bool, itemProperties map[string]any) *SchemaBuilder {
	props := b.schema["properties"].(map[string]any)
	itemSchema := map[string]any{
		"type": "object",
	}
	// Only add properties if non-nil (nil creates invalid "properties":null)
	if itemProperties != nil {
		itemSchema["properties"] = itemProperties
	}
	props[name] = map[string]any{
		"type":        "array",
		"description": description,
		"items":       itemSchema,
	}
	if required {
		b.addRequired(name)
	}
	return b
}

// addRequired adds a field to the required list, initializing it if needed
func (b *SchemaBuilder) addRequired(name string) {
	if b.schema["required"] == nil {
		b.schema["required"] = []string{}
	}
	b.schema["required"] = append(b.schema["required"].([]string), name)
}

// Build returns the completed schema
func (b *SchemaBuilder) Build() map[string]any {
	return b.schema
}

// BuildJSON returns the schema as JSON
func (b *SchemaBuilder) BuildJSON() ([]byte, error) {
	return json.Marshal(b.schema)
}
