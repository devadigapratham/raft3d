// api/models/filament.go
package models

// Filament represents a filament for 3D printing
type Filament struct {
	ID                     string `json:"id"`
	Type                   string `json:"type"` // PLA, PETG, ABS, TPU
	Color                  string `json:"color"`
	TotalWeightInGrams     int    `json:"total_weight_in_grams"`
	RemainingWeightInGrams int    `json:"remaining_weight_in_grams"`
}
