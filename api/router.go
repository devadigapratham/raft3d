// api/router.go
package api

import (
	"github.com/devadigapratham/raft3d/api/handlers"
	"github.com/devadigapratham/raft3d/raft"
	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the API routes
func SetupRouter(node *raft.Node) *gin.Engine {
	router := gin.Default()

	// Create the handler
	handler := handlers.NewHandler(node)

	// Apply middleware
	router.Use(handler.RaftLeaderMiddleware())

	// API group
	api := router.Group("/api/v1")
	{
		// Printer endpoints
		api.POST("/printers", handler.CreatePrinter)
		api.GET("/printers", handler.GetPrinters)

		// Filament endpoints
		api.POST("/filaments", handler.CreateFilament)
		api.GET("/filaments", handler.GetFilaments)

		// Print job endpoints
		api.POST("/print_jobs", handler.CreatePrintJob)
		api.GET("/print_jobs", handler.GetPrintJobs)
		api.POST("/print_jobs/:id/status", handler.UpdatePrintJobStatus)
	}

	// Add a raft status endpoint
	router.GET("/status", func(c *gin.Context) {
		isLeader := node.Leader()
		leaderAddr := node.LeaderAddress()
		state := node.State().String()

		c.JSON(200, gin.H{
			"node_id":     node.GetFSM(),
			"is_leader":   isLeader,
			"leader_addr": leaderAddr,
			"state":       state,
		})
	})

	return router
}
