package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/devadigapratham/raft3d/api/models"
	"github.com/hashicorp/raft"
)

// FSM implements the raft.FSM interface for our 3D printing application
type FSM struct {
	mu sync.RWMutex

	// Our application state
	printers  map[string]*models.Printer
	filaments map[string]*models.Filament
	printJobs map[string]*models.PrintJob
}

// NewFSM creates a new Finite State Machine for the Raft cluster
func NewFSM() *FSM {
	return &FSM{
		printers:  make(map[string]*models.Printer),
		filaments: make(map[string]*models.Filament),
		printJobs: make(map[string]*models.PrintJob),
	}
}

// Apply applies a Raft log entry to the FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Unmarshal the command
	var cmd models.Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	// Process the command based on its type
	switch cmd.Type {
	case models.AddPrinter:
		if cmd.Printer == nil {
			return fmt.Errorf("printer is nil")
		}
		f.printers[cmd.Printer.ID] = cmd.Printer
		return nil

	case models.AddFilament:
		if cmd.Filament == nil {
			return fmt.Errorf("filament is nil")
		}
		f.filaments[cmd.Filament.ID] = cmd.Filament
		return nil

	case models.AddPrintJob:
		if cmd.PrintJob == nil {
			return fmt.Errorf("print job is nil")
		}

		// Validate printer and filament exist
		if _, ok := f.printers[cmd.PrintJob.PrinterID]; !ok {
			return fmt.Errorf("printer with ID %s does not exist", cmd.PrintJob.PrinterID)
		}
		filament, ok := f.filaments[cmd.PrintJob.FilamentID]
		if !ok {
			return fmt.Errorf("filament with ID %s does not exist", cmd.PrintJob.FilamentID)
		}

		// Calculate available filament weight
		availableWeight := filament.RemainingWeightInGrams
		for _, job := range f.printJobs {
			if job.FilamentID == cmd.PrintJob.FilamentID && (job.Status == "Queued" || job.Status == "Running") {
				availableWeight -= job.PrintWeightInGrams
			}
		}

		// Check if there's enough filament
		if cmd.PrintJob.PrintWeightInGrams > availableWeight {
			return fmt.Errorf("not enough filament remaining. Available: %d g, Required: %d g",
				availableWeight, cmd.PrintJob.PrintWeightInGrams)
		}

		// Initialize status to Queued
		cmd.PrintJob.Status = "Queued"
		f.printJobs[cmd.PrintJob.ID] = cmd.PrintJob
		return nil

	case models.UpdatePrintJob:
		job, ok := f.printJobs[cmd.JobID]
		if !ok {
			return fmt.Errorf("print job with ID %s does not exist", cmd.JobID)
		}

		// Validate status transition
		if err := models.ValidateStatusChange(job.Status, cmd.NewStatus); err != nil {
			return err
		}

		// Update status
		oldStatus := job.Status
		job.Status = cmd.NewStatus

		// Reduce filament weight if job is done
		if oldStatus == "Running" && cmd.NewStatus == "Done" {
			filament, ok := f.filaments[job.FilamentID]
			if !ok {
				return fmt.Errorf("filament with ID %s does not exist", job.FilamentID)
			}
			filament.RemainingWeightInGrams -= job.PrintWeightInGrams
			if filament.RemainingWeightInGrams < 0 {
				filament.RemainingWeightInGrams = 0
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Snapshot returns a snapshot of the FSM state
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create a deep copy of the state
	printers := make(map[string]*models.Printer)
	for k, v := range f.printers {
		printer := *v
		printers[k] = &printer
	}

	filaments := make(map[string]*models.Filament)
	for k, v := range f.filaments {
		filament := *v
		filaments[k] = &filament
	}

	printJobs := make(map[string]*models.PrintJob)
	for k, v := range f.printJobs {
		job := *v
		printJobs[k] = &job
	}

	return &fsmSnapshot{
		printers:  printers,
		filaments: filaments,
		printJobs: printJobs,
	}, nil
}

// Restore restores the FSM from a snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	// Read the snapshot data
	var snapshot fsmSnapshot
	if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
		return err
	}

	// Restore the state
	f.mu.Lock()
	defer f.mu.Unlock()

	f.printers = snapshot.printers
	f.filaments = snapshot.filaments
	f.printJobs = snapshot.printJobs

	return nil
}

// GetPrinters returns all printers
func (f *FSM) GetPrinters() []*models.Printer {
	f.mu.RLock()
	defer f.mu.RUnlock()

	printers := make([]*models.Printer, 0, len(f.printers))
	for _, printer := range f.printers {
		printers = append(printers, printer)
	}
	return printers
}

// GetFilaments returns all filaments
func (f *FSM) GetFilaments() []*models.Filament {
	f.mu.RLock()
	defer f.mu.RUnlock()

	filaments := make([]*models.Filament, 0, len(f.filaments))
	for _, filament := range f.filaments {
		filaments = append(filaments, filament)
	}
	return filaments
}

// GetPrintJobs returns all print jobs
func (f *FSM) GetPrintJobs() []*models.PrintJob {
	f.mu.RLock()
	defer f.mu.RUnlock()

	printJobs := make([]*models.PrintJob, 0, len(f.printJobs))
	for _, job := range f.printJobs {
		printJobs = append(printJobs, job)
	}
	return printJobs
}

// GetPrintJobsByStatus returns print jobs filtered by status
func (f *FSM) GetPrintJobsByStatus(status string) []*models.PrintJob {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var jobs []*models.PrintJob
	for _, job := range f.printJobs {
		if job.Status == status {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// GetPrintJob returns a print job by ID
func (f *FSM) GetPrintJob(id string) (*models.PrintJob, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	job, ok := f.printJobs[id]
	return job, ok
}

// fsmSnapshot implements the raft.FSMSnapshot interface
type fsmSnapshot struct {
	printers  map[string]*models.Printer
	filaments map[string]*models.Filament
	printJobs map[string]*models.PrintJob
}

// Persist saves the snapshot to the provided sink
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode the snapshot
		if err := json.NewEncoder(sink).Encode(s); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
		return err
	}

	return nil
}

// Release is a no-op
func (s *fsmSnapshot) Release() {}
