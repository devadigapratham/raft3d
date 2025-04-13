package handlers

import (
	"net/http"

	"github.com/devadigapratham/raft3d/api/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreatePrintJob creates a new print job
func (h *Handler) CreatePrintJob(c *gin.Context) {
	var printJob models.PrintJob
	if err := c.ShouldBindJSON(&printJob); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate an ID if not provided
	if printJob.ID == "" {
		printJob.ID = uuid.New().String()
	}

	// Force status to be "Queued"
	printJob.Status = "Queued"

	// Create the command
	cmd := &models.Command{
		Type:     models.AddPrintJob,
		PrintJob: &printJob,
	}

	// Apply the command
	if err := h.Node.Apply(cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, printJob)
}

// GetPrintJobs returns all print jobs
func (h *Handler) GetPrintJobs(c *gin.Context) {
	// Check if status filter is provided
	status := c.Query("status")

	var printJobs []*models.PrintJob
	if status != "" && models.IsValidPrintJobStatus(status) {
		printJobs = h.Node.GetFSM().GetPrintJobsByStatus(status)
	} else {
		printJobs = h.Node.GetFSM().GetPrintJobs()
	}

	c.JSON(http.StatusOK, printJobs)
}

// UpdatePrintJobStatus updates the status of a print job
func (h *Handler) UpdatePrintJobStatus(c *gin.Context) {
	jobID := c.Param("id")
	newStatus := c.Query("status")

	// Validate status
	if !models.IsValidPrintJobStatus(newStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	// Check if the job exists
	_, exists := h.Node.GetFSM().GetPrintJob(jobID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "print job not found"})
		return
	}

	// Create the command
	cmd := &models.Command{
		Type:      models.UpdatePrintJob,
		JobID:     jobID,
		NewStatus: newStatus,
	}

	// Apply the command
	if err := h.Node.Apply(cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the updated job
	job, _ := h.Node.GetFSM().GetPrintJob(jobID)
	c.JSON(http.StatusOK, job)
}
