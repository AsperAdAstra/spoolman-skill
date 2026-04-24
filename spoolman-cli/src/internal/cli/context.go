package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vibecoder/spoolctl/internal/api"
	"github.com/vibecoder/spoolctl/internal/config"
	"github.com/vibecoder/spoolctl/internal/db"
)

func newContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Compact LLM context snapshot (CTXv1)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			c, err := api.NewClient(cfg, flagVerbose)
			if err != nil {
				return err
			}

			// Health check
			health := "ok"
			if _, err := c.GetHealth(); err != nil {
				health = "error"
			}

			// Info for version check
			versionNote := ""
			if info, err := c.GetInfo(); err == nil {
				if info.Version != config.TestedVersion {
					versionNote = fmt.Sprintf(" version=%s", info.Version)
				}
			}

			ts := time.Now().UTC().Format(time.RFC3339)
			fmt.Printf("CTXv1 server=%s health=%s%s ts=%s\n", cfg.ServerURL, health, versionNote, ts)

			if health != "ok" {
				fmt.Println("ERROR: server unhealthy; stopping context collection")
				return nil
			}

			// Counts
			vendors, _ := c.ListVendorsTyped()
			filaments, _ := c.ListFilamentsTyped()
			spools, _ := c.ListSpoolsTyped("", true)

			active := 0
			archived := 0
			low := 0
			const lowThreshold = 150.0 // grams

			for _, s := range spools {
				if s.Archived {
					archived++
				} else {
					active++
					if s.RemainingWeight != nil && *s.RemainingWeight < lowThreshold {
						low++
					}
				}
			}

			fmt.Printf("COUNTS vendors=%d filaments=%d spools=%d low=%d archived=%d\n",
				len(vendors), len(filaments), active, low, archived)

			// Material breakdown
			matCounts := map[string]int{}
			for _, f := range filaments {
				if f.Material != nil {
					matCounts[*f.Material]++
				} else {
					matCounts["UNKNOWN"]++
				}
			}
			fmt.Print("MATERIALS")
			keys := make([]string, 0, len(matCounts))
			for k := range matCounts {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return matCounts[keys[i]] > matCounts[keys[j]] })
			for _, k := range keys {
				fmt.Printf(" %s=%d", k, matCounts[k])
			}
			fmt.Println()

			// Low spools (non-archived, sorted by remaining weight)
			type lowSpool struct {
				id     int
				weight float64
				label  string
			}
			var lowSpools []lowSpool
			for _, s := range spools {
				if !s.Archived && s.RemainingWeight != nil && *s.RemainingWeight < lowThreshold {
					mat := ""
					if s.Filament.Material != nil {
						mat = *s.Filament.Material
					}
					name := ""
					if s.Filament.Name != nil {
						name = *s.Filament.Name
					}
					label := mat + "-" + name
					label = strings.ReplaceAll(label, " ", "-")
					lowSpools = append(lowSpools, lowSpool{
						id:     s.ID,
						weight: *s.RemainingWeight,
						label:  label,
					})
				}
			}
			sort.Slice(lowSpools, func(i, j int) bool { return lowSpools[i].weight < lowSpools[j].weight })
			if len(lowSpools) > 0 {
				parts := make([]string, 0, len(lowSpools))
				for _, ls := range lowSpools {
					if len(parts) >= 10 {
						parts = append(parts, fmt.Sprintf("...+%d", len(lowSpools)-10))
						break
					}
					parts = append(parts, fmt.Sprintf("id=%d:%.0fg:%s", ls.id, ls.weight, ls.label))
				}
				fmt.Printf("LOW_SPOOLS %s\n", strings.Join(parts, "|"))
			}

			// Recent use (last_used non-nil, sorted desc)
			type recentUse struct {
				id       int
				usedW    float64
				lastUsed string
				label    string
			}
			var recent []recentUse
			for _, s := range spools {
				if s.LastUsed != nil && s.UsedWeight > 0 {
					mat := ""
					if s.Filament.Material != nil {
						mat = *s.Filament.Material
					}
					name := ""
					if s.Filament.Name != nil {
						name = *s.Filament.Name
					}
					label := strings.ReplaceAll(mat+"-"+name, " ", "-")
					recent = append(recent, recentUse{
						id:       s.ID,
						usedW:    s.UsedWeight,
						lastUsed: *s.LastUsed,
						label:    label,
					})
				}
			}
			sort.Slice(recent, func(i, j int) bool { return recent[i].lastUsed > recent[j].lastUsed })
			if len(recent) > 0 {
				parts := make([]string, 0, 5)
				for i, r := range recent {
					if i >= 5 {
						break
					}
					day := ""
					if len(r.lastUsed) >= 10 {
						day = r.lastUsed[:10]
					}
					parts = append(parts, fmt.Sprintf("id=%d:-%.0fg:%s:%s", r.id, r.usedW, r.label, day))
				}
				fmt.Printf("RECENT_USE %s\n", strings.Join(parts, "|"))
			}

			// DB state
			cacheDir, _ := db.CacheDir()
			dbSource := "none"
			if cacheDir != "" {
				dbSource = "upstream"
			}
			fmt.Printf("DB_STATE source=%s cache_dir=%s\n", dbSource, cacheDir)

			return nil
		},
	}
}
