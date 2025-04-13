package handlers

import (
	"github.com/devadigapratham/raft3d/raft"
	"github.com/gin-gonic/gin"
)

// Handler represents the API handlers
type Handler struct {
	Node *raft.Node
}

// NewHandler creates a new Handler
func NewHandler(node *raft.Node) *Handler {
	return &Handler{
		Node: node,
	}
}

// RaftLeaderMiddleware ensures a request is forwarded to the leader
func (h *Handler) RaftLeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to write operations
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			// Check if this node is the leader
			if !h.Node.Leader() {
				// Respond with the leader's address
				c.JSON(409, gin.H{
					"error":  "not the leader",
					"leader": h.Node.LeaderAddress(),
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
