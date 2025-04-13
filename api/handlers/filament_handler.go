package handlers

import (
	"net/http"

	"github.com/devadigapratham/raft3d/api/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateFilament creates a new filament
func (h *Handler) CreateFilament(c *gin.Context) {
	var filament models.Filament
	if err := c.ShouldBindJSON(&filament); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate filament type
	if !models.IsValidFilamentType(filament.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filament type"})
		return
	}

	// Generate an ID if not provided
	if filament.ID == "" {
		filament.ID = uuid.New().String()
	}

	// If remaining weight not specified, set it to total weight
	if filament.RemainingWeightInGrams == 0 {
		filament.RemainingWeightInGrams = filament.TotalWeightInGrams
	}

	// Create the command
	cmd := &models.Command{
		Type:     models.AddFilament,
		Filament: &filament,
	}

	// Apply the command
	if err := h.Node.Apply(cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, filament)
}

// GetFilaments returns all filaments
func (h *Handler) GetFilaments(c *gin.Context) {
	filaments := h.Node.GetFSM().GetFilaments()
	c.JSON(http.StatusOK, filaments)
}
