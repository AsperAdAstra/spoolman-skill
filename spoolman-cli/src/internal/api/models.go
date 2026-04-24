package api

import "encoding/json"

// Vendor mirrors the Spoolman Vendor response schema.
type Vendor struct {
	ID             int               `json:"id"`
	Registered     string            `json:"registered"`
	Name           string            `json:"name"`
	Comment        *string           `json:"comment,omitempty"`
	EmptySpoolWeight *float64        `json:"empty_spool_weight,omitempty"`
	ExternalID     *string           `json:"external_id,omitempty"`
	Extra          map[string]string `json:"extra"`
}

type VendorCreate struct {
	Name             string            `json:"name"`
	Comment          *string           `json:"comment,omitempty"`
	EmptySpoolWeight *float64          `json:"empty_spool_weight,omitempty"`
	ExternalID       *string           `json:"external_id,omitempty"`
	Extra            map[string]string `json:"extra,omitempty"`
}

type VendorUpdate struct {
	Name             *string           `json:"name,omitempty"`
	Comment          *string           `json:"comment,omitempty"`
	EmptySpoolWeight *float64          `json:"empty_spool_weight,omitempty"`
	ExternalID       *string           `json:"external_id,omitempty"`
	Extra            map[string]string `json:"extra,omitempty"`
}

// Filament mirrors the Spoolman Filament response schema.
type Filament struct {
	ID                 int               `json:"id"`
	Registered         string            `json:"registered"`
	Name               *string           `json:"name,omitempty"`
	Vendor             *Vendor           `json:"vendor,omitempty"`
	Material           *string           `json:"material,omitempty"`
	Price              *float64          `json:"price,omitempty"`
	Density            float64           `json:"density"`
	Diameter           float64           `json:"diameter"`
	Weight             *float64          `json:"weight,omitempty"`
	SpoolWeight        *float64          `json:"spool_weight,omitempty"`
	ArticleNumber      *string           `json:"article_number,omitempty"`
	Comment            *string           `json:"comment,omitempty"`
	ExtruderTemp       *int              `json:"settings_extruder_temp,omitempty"`
	BedTemp            *int              `json:"settings_bed_temp,omitempty"`
	ColorHex           *string           `json:"color_hex,omitempty"`
	MultiColorHexes    *string           `json:"multi_color_hexes,omitempty"`
	MultiColorDirection *string          `json:"multi_color_direction,omitempty"`
	ExternalID         *string           `json:"external_id,omitempty"`
	Extra              map[string]string `json:"extra"`
}

type FilamentCreate struct {
	Name                *string           `json:"name,omitempty"`
	VendorID            *int              `json:"vendor_id,omitempty"`
	Material            *string           `json:"material,omitempty"`
	Price               *float64          `json:"price,omitempty"`
	Density             float64           `json:"density"`
	Diameter            float64           `json:"diameter"`
	Weight              *float64          `json:"weight,omitempty"`
	SpoolWeight         *float64          `json:"spool_weight,omitempty"`
	ArticleNumber       *string           `json:"article_number,omitempty"`
	Comment             *string           `json:"comment,omitempty"`
	ExtruderTemp        *int              `json:"settings_extruder_temp,omitempty"`
	BedTemp             *int              `json:"settings_bed_temp,omitempty"`
	ColorHex            *string           `json:"color_hex,omitempty"`
	MultiColorHexes     *string           `json:"multi_color_hexes,omitempty"`
	MultiColorDirection *string           `json:"multi_color_direction,omitempty"`
	ExternalID          *string           `json:"external_id,omitempty"`
	Extra               map[string]string `json:"extra,omitempty"`
}

type FilamentUpdate struct {
	Name                *string           `json:"name,omitempty"`
	VendorID            *int              `json:"vendor_id,omitempty"`
	Material            *string           `json:"material,omitempty"`
	Price               *float64          `json:"price,omitempty"`
	Density             *float64          `json:"density,omitempty"`
	Diameter            *float64          `json:"diameter,omitempty"`
	Weight              *float64          `json:"weight,omitempty"`
	SpoolWeight         *float64          `json:"spool_weight,omitempty"`
	ArticleNumber       *string           `json:"article_number,omitempty"`
	Comment             *string           `json:"comment,omitempty"`
	ExtruderTemp        *int              `json:"settings_extruder_temp,omitempty"`
	BedTemp             *int              `json:"settings_bed_temp,omitempty"`
	ColorHex            *string           `json:"color_hex,omitempty"`
	MultiColorHexes     *string           `json:"multi_color_hexes,omitempty"`
	MultiColorDirection *string           `json:"multi_color_direction,omitempty"`
	ExternalID          *string           `json:"external_id,omitempty"`
	Extra               map[string]string `json:"extra,omitempty"`
}

// Spool mirrors the Spoolman Spool response schema.
type Spool struct {
	ID              int               `json:"id"`
	Registered      string            `json:"registered"`
	FirstUsed       *string           `json:"first_used,omitempty"`
	LastUsed        *string           `json:"last_used,omitempty"`
	Filament        Filament          `json:"filament"`
	Price           *float64          `json:"price,omitempty"`
	RemainingWeight *float64          `json:"remaining_weight,omitempty"`
	InitialWeight   *float64          `json:"initial_weight,omitempty"`
	SpoolWeight     *float64          `json:"spool_weight,omitempty"`
	UsedWeight      float64           `json:"used_weight"`
	RemainingLength *float64          `json:"remaining_length,omitempty"`
	UsedLength      float64           `json:"used_length"`
	Location        *string           `json:"location,omitempty"`
	LotNr           *string           `json:"lot_nr,omitempty"`
	Comment         *string           `json:"comment,omitempty"`
	Archived        bool              `json:"archived"`
	Extra           map[string]string `json:"extra"`
}

type SpoolCreate struct {
	FilamentID      int               `json:"filament_id"`
	Price           *float64          `json:"price,omitempty"`
	InitialWeight   *float64          `json:"initial_weight,omitempty"`
	SpoolWeight     *float64          `json:"spool_weight,omitempty"`
	RemainingWeight *float64          `json:"remaining_weight,omitempty"`
	UsedWeight      *float64          `json:"used_weight,omitempty"`
	Location        *string           `json:"location,omitempty"`
	LotNr           *string           `json:"lot_nr,omitempty"`
	Comment         *string           `json:"comment,omitempty"`
	Archived        bool              `json:"archived,omitempty"`
	Extra           map[string]string `json:"extra,omitempty"`
}

type SpoolUpdate struct {
	FilamentID      *int              `json:"filament_id,omitempty"`
	Price           *float64          `json:"price,omitempty"`
	InitialWeight   *float64          `json:"initial_weight,omitempty"`
	SpoolWeight     *float64          `json:"spool_weight,omitempty"`
	RemainingWeight *float64          `json:"remaining_weight,omitempty"`
	UsedWeight      *float64          `json:"used_weight,omitempty"`
	Location        *string           `json:"location,omitempty"`
	LotNr           *string           `json:"lot_nr,omitempty"`
	Comment         *string           `json:"comment,omitempty"`
	Archived        *bool             `json:"archived,omitempty"`
	Extra           map[string]string `json:"extra,omitempty"`
}

type SpoolUse struct {
	UseWeight *float64 `json:"use_weight,omitempty"`
	UseLength *float64 `json:"use_length,omitempty"`
}

type SpoolMeasure struct {
	Weight float64 `json:"weight"`
}

// ExternalFilament is a SpoolmanDB filament record.
type ExternalFilament struct {
	ID                  string   `json:"id"`
	Manufacturer        string   `json:"manufacturer"`
	Name                string   `json:"name"`
	Material            string   `json:"material"`
	Density             float64  `json:"density"`
	Weight              float64  `json:"weight"`
	SpoolWeight         *float64 `json:"spool_weight,omitempty"`
	SpoolType           *string  `json:"spool_type,omitempty"`
	Diameter            float64  `json:"diameter"`
	ColorHex            *string  `json:"color_hex,omitempty"`
	ColorHexes          []string `json:"color_hexes,omitempty"`
	ExtruderTemp        *int     `json:"extruder_temp,omitempty"`
	BedTemp             *int     `json:"bed_temp,omitempty"`
	Finish              *string  `json:"finish,omitempty"`
	MultiColorDirection *string  `json:"multi_color_direction,omitempty"`
	Pattern             *string  `json:"pattern,omitempty"`
	Translucent         bool     `json:"translucent"`
	Glow                bool     `json:"glow"`
}

// ExternalMaterial is a SpoolmanDB material record.
type ExternalMaterial struct {
	Material    string   `json:"material"`
	Density     float64  `json:"density"`
	ExtruderTemp *int    `json:"extruder_temp,omitempty"`
	BedTemp     *int     `json:"bed_temp,omitempty"`
}

// Info is the Spoolman server info.
type Info struct {
	Version          string  `json:"version"`
	DebugMode        bool    `json:"debug_mode"`
	AutomaticBackups bool    `json:"automatic_backups"`
	DataDir          string  `json:"data_dir"`
	LogsDir          string  `json:"logs_dir"`
	BackupsDir       string  `json:"backups_dir"`
	DBType           string  `json:"db_type"`
	GitCommit        *string `json:"git_commit,omitempty"`
	BuildDate        *string `json:"build_date,omitempty"`
}

// HealthCheck is the Spoolman health response.
type HealthCheck struct {
	Status string `json:"status"`
}

// PrettyJSON returns indented JSON bytes.
func PrettyJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
