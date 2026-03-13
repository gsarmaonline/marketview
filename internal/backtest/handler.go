package backtest

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// StrategyConfig is the strategy field in a backtest request.
type StrategyConfig struct {
	Name string `json:"name"`
}

// Request is the body for POST /api/backtest.
type Request struct {
	Symbol   string         `json:"symbol"   binding:"required"`
	From     string         `json:"from"     binding:"required"` // YYYY-MM-DD
	To       string         `json:"to"       binding:"required"` // YYYY-MM-DD
	Capital  float64        `json:"capital"  binding:"required"`
	Strategy StrategyConfig `json:"strategy" binding:"required"`
}

// registry maps strategy names to implementations.
// Add new strategies here as they are built.
var registry = map[string]Strategy{
	"buy_and_hold": BuyAndHold{},
}

// Handler handles backtest HTTP requests.
type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Handle(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Symbol = strings.ToUpper(strings.TrimSpace(req.Symbol))

	strategy, ok := registry[req.Strategy.Name]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown strategy: " + req.Strategy.Name})
		return
	}

	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date, use YYYY-MM-DD"})
		return
	}
	to, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date, use YYYY-MM-DD"})
		return
	}
	if !to.After(from) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to must be after from"})
		return
	}

	prices, err := FetchHistory(req.Symbol, from, to)
	if err != nil {
		log.Printf("backtest history fetch error for %s: %v", req.Symbol, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch historical data: " + err.Error()})
		return
	}
	if len(prices) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "no price data found for the given range"})
		return
	}

	result := Run(req.Symbol, prices, strategy, req.Capital)
	c.JSON(http.StatusOK, result)
}
