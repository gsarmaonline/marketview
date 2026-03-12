package deepresearch

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler exposes deep research HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a new deep research Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleDeepResearch handles GET /api/stock/:symbol/deep-research.
func (h *Handler) HandleDeepResearch(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	result, err := h.service.Fetch(symbol)
	if err != nil {
		log.Printf("deep research error for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch deep research data"})
		return
	}

	c.JSON(http.StatusOK, result)
}
