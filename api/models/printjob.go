// api/models/printjob.go
package models

// PrintJob represents a 3D printing job
type PrintJob struct {
	ID                 string `json:"id"`
	PrinterID          string `json:"printer_id"`
	FilamentID         string `json:"filament_id"`
	Filepath           string `json:"filepath"`
	PrintWeightInGrams int    `json:"print_weight_in_grams"`
	Status             string `json:"status"` // Queued, Running, Done, Canceled
}
