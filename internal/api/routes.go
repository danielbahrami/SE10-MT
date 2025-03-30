package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(router *gin.Engine, dbpool *pgxpool.Pool) {
	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Query endpoint
	router.POST("/query", func(c *gin.Context) {
		// Authenticate the user.
		user, err := AuthenticateUser(c, dbpool)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// User is now authenticated
		var req struct {
			Cypher string `json:"cypher"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "User authenticated and query processed",
			"user":    user.Email,
			"cypher":  req.Cypher,
		})
	})
}
