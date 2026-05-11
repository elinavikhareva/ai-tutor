package gemini

import (
	"maps"
	"slices"
)

// Schema is the subset of OpenAPI schema that Gemini accepts as responseSchema.
type Schema struct {
	Type       string             `json:"type"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Enum       []string           `json:"enum,omitempty"`
}

// Object makes an object schema where all properties except optional are required.
func Object(props map[string]*Schema, optional ...string) *Schema {
	var required []string
	for _, name := range slices.Sorted(maps.Keys(props)) {
		if !slices.Contains(optional, name) {
			required = append(required, name)
		}
	}
	return &Schema{Type: "OBJECT", Properties: props, Required: required}
}

func Array(items *Schema) *Schema { return &Schema{Type: "ARRAY", Items: items} }

func String() *Schema { return &Schema{Type: "STRING"} }

func Integer() *Schema { return &Schema{Type: "INTEGER"} }

func Enum(values ...string) *Schema { return &Schema{Type: "STRING", Enum: values} }
