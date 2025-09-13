package provider

import (
	jsoniter "encoding/json"

	"github.com/invopop/jsonschema"
)

// MarshalSchema indents the schema to JSON bytes.
func MarshalSchema(sch *jsonschema.Schema) ([]byte, error) {
	return jsoniter.MarshalIndent(sch, "", "  ")
}

// CatalogV2Schema returns a JSON Schema for the v2 provider catalog.
// Shape: top-level object with provider IDs as keys, each a Provider object.
func CatalogV2Schema() *jsonschema.Schema {
	r := jsonschema.Reflector{ExpandedStruct: true}
	providerSch := r.Reflect(&Provider{})
	top := &jsonschema.Schema{
		Title:                "codectl provider catalog (v2)",
		Description:          "Top-level map of providers (keys: provider IDs).",
		Type:                 "object",
		AdditionalProperties: providerSch,
	}
	return top
}
