package handlers

import (
	"net/http"

	"github.com/devadigapratham/raft3d/api/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreatePrinter creates a new printer
func (h *Handler) CreatePrinter(c *gin.Context) {
	var printer models.Printer
	if err := c.ShouldBindJSON(&printer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate an ID if not provided
	if printer.ID == "" {
		printer.ID = uuid.New().String()
	}

	// Create the command
	cmd := &models.Command{
		Type:    models.AddPrinter,
		Printer: &printer,
	}

	// Apply the command
	if err := h.Node.Apply(cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, printer)
}

// GetPrinters returns all printers
func (h *Handler) GetPrinters(c *gin.Context) {
	printers := h.Node.GetFSM().GetPrinters()
	c.JSON(http.StatusOK, printers)
}
