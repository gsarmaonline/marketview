package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
)

// Server wires up all HTTP routes and their dependencies.
type Server struct {
	router     *gin.Engine
	indicators []indicators.Indicator
	mfHandler  *mutualfund.Handler
}

func New(inds []indicators.Indicator, mfHandler *mutualfund.Handler) *Server {
	r := gin.Default()

	s := &Server{router: r, indicators: inds, mfHandler: mfHandler}

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/api/indicators", s.handleIndicators)
	r.GET("/api/news", s.handleNews)
	r.GET("/api/mutual-fund/search", mfHandler.HandleSearch)
	r.GET("/api/mutual-fund/:schemeCode", mfHandler.HandleDetails)

	return s
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) handleNews(c *gin.Context) {
	items, err := news.Fetch(20)
	if err != nil {
		log.Printf("error fetching news: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch news"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleIndicators(c *gin.Context) {
	results := make([]indicators.IndicatorResult, 0, len(s.indicators))
	for _, ind := range s.indicators {
		result, err := ind.Fetch()
		if err != nil {
			c.Error(err)
			continue
		}
		results = append(results, result)
	}
	c.JSON(http.StatusOK, results)
}
