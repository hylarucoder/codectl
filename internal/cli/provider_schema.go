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
	Long:  "输出 v2 结构（providers 映射 + 可选 mcp 数组）的 JSON Schema，用于校验 ~/.codectl/provider.json。",
	RunE: func(cmd *cobra.Command, args []string) error {
		sch := provider.CatalogV2Schema()
		b, err := provider.MarshalSchema(sch)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	},
}
