package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibecoder/spoolctl/internal/api"
)

const (
	upstreamFilamentsURL = "https://donkie.github.io/SpoolmanDB/filaments.json"
	upstreamMaterialsURL = "https://donkie.github.io/SpoolmanDB/materials.json"
	cacheFilaments       = "filaments.json"
	cacheMaterials       = "materials.json"
)

// Source indicates where data came from.
type Source string

const (
	SourceCache   Source = "cache"
	SourceNetwork Source = "network"
)

// DB holds loaded SpoolmanDB data.
type DB struct {
	Filaments []api.ExternalFilament
	Materials []api.ExternalMaterial
	Source    Source
	FetchedAt time.Time
}

// CacheDir returns the spoolctl cache directory.
func CacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "spoolctl"), nil
}

// Load loads SpoolmanDB from cache (if present) or network.
// forceRefresh bypasses cache and always fetches from network.
func Load(forceRefresh bool, verbose bool) (*DB, error) {
	dir, err := CacheDir()
	if err != nil {
		return nil, err
	}
	fPath := filepath.Join(dir, cacheFilaments)
	mPath := filepath.Join(dir, cacheMaterials)

	if !forceRefresh {
		if f, err := os.ReadFile(fPath); err == nil {
			if m, err := os.ReadFile(mPath); err == nil {
				db, err := parseDB(f, m, SourceCache)
				if err == nil {
					if verbose {
						info, _ := os.Stat(fPath)
						age := time.Since(info.ModTime()).Truncate(time.Minute)
						fmt.Fprintf(os.Stderr, "# db source=cache age=%s\n", age)
					}
					return db, nil
				}
			}
		}
	}

	// Fetch from network
	if verbose {
		fmt.Fprintln(os.Stderr, "# db source=network fetching SpoolmanDB...")
	}
	fc, err := fetchURL(upstreamFilamentsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching filaments: %w", err)
	}
	mc, err := fetchURL(upstreamMaterialsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching materials: %w", err)
	}

	db, err := parseDB(fc, mc, SourceNetwork)
	if err != nil {
		return nil, err
	}

	// Persist to cache
	if err := os.MkdirAll(dir, 0755); err == nil {
		_ = os.WriteFile(fPath, fc, 0644)
		_ = os.WriteFile(mPath, mc, 0644)
	}
	return db, nil
}

func parseDB(filBytes, matBytes []byte, src Source) (*DB, error) {
	var filaments []api.ExternalFilament
	if err := json.Unmarshal(filBytes, &filaments); err != nil {
		return nil, fmt.Errorf("parsing filaments: %w", err)
	}
	var materials []api.ExternalMaterial
	if err := json.Unmarshal(matBytes, &materials); err != nil {
		return nil, fmt.Errorf("parsing materials: %w", err)
	}
	return &DB{
		Filaments: filaments,
		Materials: materials,
		Source:    src,
		FetchedAt: time.Now(),
	}, nil
}

func fetchURL(rawURL string) ([]byte, error) {
	return fetchHTTP(rawURL)
}

// FilterFilaments returns filaments matching the given filters.
func (db *DB) FilterFilaments(manufacturer, material string, diameter float64) []api.ExternalFilament {
	var out []api.ExternalFilament
	for _, f := range db.Filaments {
		if manufacturer != "" && !strings.EqualFold(f.Manufacturer, manufacturer) {
			continue
		}
		if material != "" && !strings.EqualFold(f.Material, material) {
			continue
		}
		if diameter > 0 && abs(f.Diameter-diameter) > 0.01 {
			continue
		}
		out = append(out, f)
	}
	return out
}

// Lookup returns the filament with the given ID, or nil.
func (db *DB) Lookup(id string) *api.ExternalFilament {
	for i := range db.Filaments {
		if db.Filaments[i].ID == id {
			return &db.Filaments[i]
		}
	}
	return nil
}

// FindMaterial returns the material record for the given material name (case-insensitive).
func (db *DB) FindMaterial(name string) *api.ExternalMaterial {
	for i := range db.Materials {
		if strings.EqualFold(db.Materials[i].Material, name) {
			return &db.Materials[i]
		}
	}
	return nil
}

// FilamentToCreate converts an ExternalFilament to a FilamentCreate, optionally overriding vendor_id.
func FilamentToCreate(ef *api.ExternalFilament, vendorID *int) api.FilamentCreate {
	fc := api.FilamentCreate{
		Name:        &ef.Name,
		Material:    &ef.Material,
		Density:     ef.Density,
		Diameter:    ef.Diameter,
		Weight:      &ef.Weight,
		SpoolWeight: ef.SpoolWeight,
		ExternalID:  &ef.ID,
		VendorID:    vendorID,
	}
	if ef.ExtruderTemp != nil {
		fc.ExtruderTemp = ef.ExtruderTemp
	}
	if ef.BedTemp != nil {
		fc.BedTemp = ef.BedTemp
	}
	if ef.ColorHex != nil {
		fc.ColorHex = ef.ColorHex
	}
	if len(ef.ColorHexes) > 0 {
		joined := strings.Join(ef.ColorHexes, ",")
		fc.MultiColorHexes = &joined
	}
	if ef.MultiColorDirection != nil {
		fc.MultiColorDirection = ef.MultiColorDirection
	}
	return fc
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
