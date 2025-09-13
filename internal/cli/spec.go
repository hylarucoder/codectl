package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"codectl/internal/specui"
	"codectl/internal/tools"
	appver "codectl/internal/version"
)

func init() {
	rootCmd.AddCommand(specCmd)
}

type specTool struct {
	ID        tools.ToolID `json:"id"`
	Name      string       `json:"name"`
	Installed bool         `json:"installed"`
	Version   string       `json:"version,omitempty"`
	Latest    string       `json:"latest,omitempty"`
	Source    string       `json:"source,omitempty"`
	Error     string       `json:"error,omitempty"`
}

type specInfo struct {
	CodectlVersion string     `json:"codectlVersion"`
	OS             string     `json:"os"`
	Arch           string     `json:"arch"`
	Tools          []specTool `json:"tools"`
}

var (
	specPretty bool
	specJSON   bool
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Open Spec UI (default) or dump JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !specJSON {
			// Default: open interactive UI
			return specui.Start()
		}
		out := specInfo{
			CodectlVersion: appver.AppVersion,
			OS:             runtime.GOOS,
			Arch:           runtime.GOARCH,
			Tools:          make([]specTool, 0, len(tools.Tools)),
		}
		for _, t := range tools.Tools {
			res := tools.CheckTool(t)
			out.Tools = append(out.Tools, specTool{
				ID:        t.ID,
				Name:      t.DisplayName,
				Installed: res.Installed,
				Version:   res.Version,
				Latest:    res.Latest,
				Source:    res.Source,
				Error:     res.Err,
			})
		}
		if specPretty {
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		b, err := json.Marshal(out)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	specCmd.Flags().BoolVar(&specPretty, "pretty", true, "pretty-print JSON output")
	specCmd.Flags().BoolVarP(&specJSON, "json", "j", false, "print environment spec to JSON instead of UI")
}
