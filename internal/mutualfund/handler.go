package mutualfund

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler handles mutual fund HTTP requests.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleSearch handles GET /api/mutual-fund/search?q={query}
func (h *Handler) HandleSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing query parameter 'q'"})
		return
	}

	results, err := h.service.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}

// AnalysePortfolio delegates to the service layer.
func (h *Handler) AnalysePortfolio(inputs []FundInput) (*PortfolioAnalysis, error) {
	return h.service.AnalysePortfolio(inputs)
}

// HandleDetails handles GET /api/mutual-fund/:schemeCode
func (h *Handler) HandleDetails(c *gin.Context) {
	schemeCode, err := strconv.Atoi(c.Param("schemeCode"))
	if err != nil || schemeCode <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheme code"})
		return
	}

	details, err := h.service.GetDetails(schemeCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, details)
}
