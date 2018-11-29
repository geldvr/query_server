package routes

import (
	"github.com/gin-gonic/gin"
	"itv/query-server/handlers"
	"os"
)

func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	if os.Getenv("LOGGER") == "1" {
		r.Use(gin.Logger())
	}

	r.Use(gin.Recovery())

	g := r.Group("/api")
	{
		g.GET("/queries/:id", handlers.GetQueryById)
		g.GET("/queries", handlers.GetQueries)
		g.POST("/queries", handlers.CreateQuery)
		g.DELETE("/queries/:id", handlers.DeleteQueryById)
	}

	return r
}
