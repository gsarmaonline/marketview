package api

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"marketview/internal/deepresearch"
	"marketview/internal/indicators"
	"marketview/internal/mutualfund"
	"marketview/internal/news"
	"marketview/internal/portfolio"
	"marketview/internal/stock"
)

// Server wires up all HTTP routes and their dependencies.
type Server struct {
	router     *gin.Engine
	indicators []indicators.Indicator
	mfHandler  *mutualfund.Handler
	drHandler  *deepresearch.Handler
	newsStore  *news.Store
	shutdown   func()
}

func New(ctx context.Context, pool *pgxpool.Pool, inds []indicators.Indicator, mfHandler *mutualfund.Handler, newsStore *news.Store, drHandler *deepresearch.Handler) (*Server, error) {
	shutdown := func() {}
	if pool != nil {
		shutdown = pool.Close
	}

	r := gin.Default()
	s := &Server{
		router:     r,
		indicators: inds,
		mfHandler:  mfHandler,
		drHandler:  drHandler,
		newsStore:  newsStore,
		shutdown:   shutdown,
	}

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/api/indicators", s.handleIndicators)
	r.GET("/api/news", s.handleNews)
	r.GET("/api/news/stock/:symbol", s.handleStockNews)
	r.GET("/api/mutual-fund/search", mfHandler.HandleSearch)
	r.GET("/api/mutual-fund/:schemeCode", mfHandler.HandleDetails)
	r.GET("/api/stock/:symbol/deep-research", drHandler.HandleDeepResearch)
	r.GET("/api/stock/:symbol/price", s.handleStockPrice)

	// Portfolio routes (requires a DB pool)
	if pool != nil {
		repo := portfolio.NewRepository(pool)
		ph := portfolio.NewHandler(repo)

		portfolioMux := http.NewServeMux()
		ph.Register(portfolioMux)

		r.Any("/api/portfolio/holdings", gin.WrapH(portfolioMux))
		r.Any("/api/portfolio/holdings/*path", gin.WrapH(portfolioMux))
	}

	return s, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) Shutdown() {
	s.shutdown()
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

func (s *Server) handleStockNews(c *gin.Context) {
	symbol := c.Param("symbol")
	items := s.newsStore.Get(symbol)
	c.JSON(http.StatusOK, items)
}

func (s *Server) handleStockPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	result, err := stock.FetchPrice(symbol)
	if err != nil {
		log.Printf("error fetching price for %s: %v", symbol, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
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
