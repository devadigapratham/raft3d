// api/models/models.go
package models

import (
	"encoding/json"
	"errors"
	"strings"
)

// CommandType represents the type of command to be executed
type CommandType string

const (
	AddPrinter     CommandType = "ADD_PRINTER"
	AddFilament    CommandType = "ADD_FILAMENT"
	AddPrintJob    CommandType = "ADD_PRINT_JOB"
	UpdatePrintJob CommandType = "UPDATE_PRINT_JOB"
)

// Command represents a command to be applied to the FSM
type Command struct {
	Type      CommandType `json:"type"`
	Printer   *Printer    `json:"printer,omitempty"`
	Filament  *Filament   `json:"filament,omitempty"`
	PrintJob  *PrintJob   `json:"print_job,omitempty"`
	JobID     string      `json:"job_id,omitempty"`
	NewStatus string      `json:"new_status,omitempty"`
}

// Marshal serializes a command to JSON
func (c *Command) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// Unmarshal deserializes a command from JSON
func UnmarshalCommand(data []byte) (*Command, error) {
	var c Command
	err := json.Unmarshal(data, &c)
	return &c, err
}

// ValidateStatus checks if a status transition is valid
func ValidateStatusChange(currentStatus, newStatus string) error {
	switch currentStatus {
	case "Queued":
		if newStatus != "Running" && newStatus != "Canceled" {
			return errors.New("a job can only transition from Queued to Running or Canceled")
		}
	case "Running":
		if newStatus != "Done" && newStatus != "Canceled" {
			return errors.New("a job can only transition from Running to Done or Canceled")
		}
	default:
		return errors.New("invalid status transition")
	}
	return nil
}

// IsValidFilamentType checks if a filament type is valid
func IsValidFilamentType(filamentType string) bool {
	validTypes := []string{"PLA", "PETG", "ABS", "TPU"}
	upperType := strings.ToUpper(filamentType)

	for _, vt := range validTypes {
		if upperType == vt {
			return true
		}
	}
	return false
}

// IsValidPrintJobStatus checks if a print job status is valid
func IsValidPrintJobStatus(status string) bool {
	validStatuses := []string{"Queued", "Running", "Done", "Canceled"}

	for _, vs := range validStatuses {
		if status == vs {
			return true
		}
	}
	return false
}
