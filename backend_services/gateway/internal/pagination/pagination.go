package pagination

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	MaxLimit     = 100
	DefaultLimit = 20
	DefaultPage  = 1
)

type Params struct {
	Page  int
	Limit int
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

func ParsePageLimit(c *gin.Context) Params {
	page := DefaultPage
	limit := DefaultLimit

	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	return Params{
		Page:  page,
		Limit: NormalizeLimit(limit),
	}
}

func NewMeta(total, page, limit int) Meta {
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}
	return Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
