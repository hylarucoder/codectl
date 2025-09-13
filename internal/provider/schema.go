package provider

import (
    jsoniter "encoding/json"

    "github.com/invopop/jsonschema"
)

// CatalogSchema returns the JSON Schema for the provider Catalog.
// It includes basic title/description metadata for better UX.
func CatalogSchema() *jsonschema.Schema {
    r := jsonschema.Reflector{
        ExpandedStruct: true,
    }
    sch := r.Reflect(&Catalog{})
    // best-effort metadata
    sch.Title = "codectl provider catalog"
    sch.Description = "Provider catalog used by codectl (~/.codectl/provider.json). Contains remote model and MCP lists."
    return sch
}

// MarshalSchema indents the schema to JSON bytes.
func MarshalSchema(sch *jsonschema.Schema) ([]byte, error) {
    return jsoniter.MarshalIndent(sch, "", "  ")
}

