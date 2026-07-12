package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
}

func notFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, gin.H{"error": msg})
}

func conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, gin.H{"error": msg})
}

func forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, gin.H{"error": msg})
}

func internalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func clientIP(c *gin.Context) string {
	return c.ClientIP()
}
