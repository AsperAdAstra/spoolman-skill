package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/api"
	"github.com/vibecoder/spoolctl/internal/db"
)

func newDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "SpoolmanDB commands (filament community database)",
	}
	cmd.AddCommand(
		newDBFilamentsCmd(),
		newDBMaterialsCmd(),
		newDBLookupCmd(),
		newDBValidateCmd(),
		newDBDiffCmd(),
		newDBRefreshCmd(),
	)
	return cmd
}

func newDBFilamentsCmd() *cobra.Command {
	var manufacturer, material string
	var diameter float64
	var source string
	cmd := &cobra.Command{
		Use:   "filaments",
		Short: "List SpoolmanDB filaments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "spoolman" {
				c, err := newClient()
				if err != nil {
					return err
				}
				raw, err := c.ListExternalFilaments(manufacturer, material, diameter)
				if err != nil {
					return err
				}
				printRaw(raw)
				return nil
			}
			// upstream (cache or network)
			spoolDB, err := db.Load(false, flagVerbose)
			if err != nil {
				return err
			}
			results := spoolDB.FilterFilaments(manufacturer, material, diameter)
			return printJSON(results)
		},
	}
	cmd.Flags().StringVar(&manufacturer, "manufacturer", "", "Filter by manufacturer")
	cmd.Flags().StringVar(&material, "material", "", "Filter by material")
	cmd.Flags().Float64Var(&diameter, "diameter", 0, "Filter by diameter (mm)")
	cmd.Flags().StringVar(&source, "source", "upstream", "Data source: upstream or spoolman")
	return cmd
}

func newDBMaterialsCmd() *cobra.Command {
	var source string
	cmd := &cobra.Command{
		Use:   "materials",
		Short: "List SpoolmanDB materials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "spoolman" {
				c, err := newClient()
				if err != nil {
					return err
				}
				raw, err := c.ListExternalMaterials()
				if err != nil {
					return err
				}
				printRaw(raw)
				return nil
			}
			spoolDB, err := db.Load(false, flagVerbose)
			if err != nil {
				return err
			}
			return printJSON(spoolDB.Materials)
		},
	}
	cmd.Flags().StringVar(&source, "source", "upstream", "Data source: upstream or spoolman")
	return cmd
}

func newDBLookupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lookup <spoolmandb-id>",
		Short: "Look up a single SpoolmanDB filament record",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spoolDB, err := db.Load(false, flagVerbose)
			if err != nil {
				return err
			}
			ef := spoolDB.Lookup(args[0])
			if ef == nil {
				return fmt.Errorf("record not found: %s", args[0])
			}
			return printJSON(ef)
		},
	}
}

func newDBValidateCmd() *cobra.Command {
	var filePath string
	var strict bool
	cmd := &cobra.Command{
		Use:   "validate --file <spec.toml|spec.json>",
		Short: "Validate a filament spec against SpoolmanDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("--file is required")
			}
			spec, err := db.LoadSpec(filePath)
			if err != nil {
				return fmt.Errorf("loading spec: %w", err)
			}
			spoolDB, err := db.Load(false, flagVerbose)
			if err != nil {
				return err
			}
			report := db.Validate(spoolDB, spec, filePath, strict)
			return printJSON(report)
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Spec file path (TOML or JSON)")
	cmd.Flags().BoolVar(&strict, "strict", false, "Treat warnings as errors")
	return cmd
}

func newDBDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Show drift between Spoolman's built-in SpoolmanDB and upstream",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}

			// Fetch server-side external filaments
			serverRaw, err := c.ListExternalFilaments("", "", 0)
			if err != nil {
				return fmt.Errorf("fetching server external filaments: %w", err)
			}
			var serverFilaments []api.ExternalFilament
			if err := json.Unmarshal(serverRaw, &serverFilaments); err != nil {
				return err
			}
			serverByID := map[string]api.ExternalFilament{}
			for _, f := range serverFilaments {
				serverByID[f.ID] = f
			}

			// Fetch upstream
			upstreamDB, err := db.Load(false, flagVerbose)
			if err != nil {
				return err
			}
			upstreamByID := map[string]api.ExternalFilament{}
			for _, f := range upstreamDB.Filaments {
				upstreamByID[f.ID] = f
			}

				var diffs []diffEntry

			// Records in upstream but not server
			for id := range upstreamByID {
				if _, ok := serverByID[id]; !ok {
					diffs = append(diffs, diffEntry{ID: id, Status: "upstream_only"})
				}
			}
			// Records in server but not upstream
			for id := range serverByID {
				if _, ok := upstreamByID[id]; !ok {
					diffs = append(diffs, diffEntry{ID: id, Status: "server_only"})
				}
			}

			result := map[string]interface{}{
				"server_count":   len(serverFilaments),
				"upstream_count": len(upstreamDB.Filaments),
				"upstream_only":  countByStatus(diffs, "upstream_only"),
				"server_only":    countByStatus(diffs, "server_only"),
				"diffs":          diffs,
			}
			_ = os.Stderr
			return printJSON(result)
		},
	}
}

func newDBRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Force re-fetch of SpoolmanDB cache from upstream",
		RunE: func(cmd *cobra.Command, args []string) error {
			spoolDB, err := db.Load(true, true) // forceRefresh=true, verbose=true
			if err != nil {
				return err
			}
			fmt.Printf("refreshed: %d filaments, %d materials\n",
				len(spoolDB.Filaments), len(spoolDB.Materials))
			return nil
		},
	}
}

func countByStatus(diffs []diffEntry, status string) int {
	n := 0
	for _, d := range diffs {
		if d.Status == status {
			n++
		}
	}
	return n
}

type diffEntry struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}
