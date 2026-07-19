package handler

import (
	"net/http"
	"strings"

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

// validationError returns a 400 response with a user-friendly validation message.
// It sanitises Gin's raw binding error output.
func validationError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    "VALIDATION_ERROR",
		"message": sanitizeValidationError(err),
		"status":  http.StatusBadRequest,
	})
}

// sanitizeValidationError turns Gin's internal validator output into a
// human-readable message.  Example input:
//
//	"Key: 'SignupRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"
//
// becomes: "email is required"
func sanitizeValidationError(err error) string {
	msg := err.Error()

	// Handle multiple errors separated by newlines — take the first one.
	if idx := strings.Index(msg, "\n"); idx != -1 {
		msg = msg[:idx]
	}

	// Try to extract field name and tag from Gin's format.
	// Format: Key: '<Struct>.<Field>' Error:Field validation for '<Field>' failed on the '<tag>' tag
	if strings.Contains(msg, "Error:Field validation for") {
		parts := strings.SplitN(msg, "Error:Field validation for '", 2)
		if len(parts) == 2 {
			rest := parts[1]
			fieldEnd := strings.Index(rest, "'")
			if fieldEnd > 0 {
				field := rest[:fieldEnd]
				field = toSnakeCase(field)

				if strings.Contains(rest, "'required'") {
					return field + " is required"
				}
				if strings.Contains(rest, "'email'") {
					return field + " must be a valid email address"
				}
				if strings.Contains(rest, "'min=") {
					// Extract min value
					minIdx := strings.Index(rest, "'min=")
					if minIdx >= 0 {
						minRest := rest[minIdx+5:]
						if end := strings.Index(minRest, "'"); end > 0 {
							return field + " must be at least " + minRest[:end] + " characters"
						}
					}
				}
				return field + " is invalid"
			}
		}
	}

	return msg
}

// toSnakeCase converts PascalCase or camelCase to snake_case.
func toSnakeCase(s string) string {
	var result []byte
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(r+'a'-'A'))
		} else {
			result = append(result, byte(r))
		}
	}
	return string(result)
}
