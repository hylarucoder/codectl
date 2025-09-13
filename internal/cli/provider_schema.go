package cli

import (
    "fmt"

    "github.com/spf13/cobra"

    "codectl/internal/provider"
)

func init() {
    providerCmd.AddCommand(providerSchemaCmd)
}

var providerSchemaCmd = &cobra.Command{
    Use:   "schema",
    Short: "输出 provider.json 的 JSON Schema",
    Long:  "将 provider.Catalog 的 JSON Schema 输出到标准输出，用于校验 ~/.codectl/provider.json。",
    RunE: func(cmd *cobra.Command, args []string) error {
        sch := provider.CatalogSchema()
        b, err := provider.MarshalSchema(sch)
        if err != nil {
            return err
        }
        fmt.Println(string(b))
        return nil
    },
}

