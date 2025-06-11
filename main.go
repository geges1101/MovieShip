package main

import (
	"log"
	"os"

	"movieship/internal"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	db, err := internal.InitDB()
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	oidc, err := internal.NewOIDCMiddleware()
	if err != nil {
		log.Fatalf("OIDC error: %v", err)
	}
	h := &internal.Handlers{DB: db}

	r := gin.Default()

	// Add CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:8080"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Add debug logging middleware
	r.Use(func(c *gin.Context) {
		log.Printf("Request: %s %s", c.Request.Method, c.Request.URL.Path)
		log.Printf("Authorization header: %v", c.GetHeader("Authorization") != "")
		c.Next()
		log.Printf("Response status: %d", c.Writer.Status())
	})

	r.GET("/api/movies", oidc.Middleware(), h.ListMovies)
	r.GET("/api/movies/:id", oidc.Middleware(), h.GetMovie)
	r.GET("/api/history", oidc.Middleware(), h.GetWatchHistory)
	r.POST("/api/history", oidc.Middleware(), h.UpdateWatchHistory)
	r.POST("/api/upload", oidc.Middleware(), h.UploadVideo)
	r.GET("/api/movies/:id/stream", oidc.Middleware(), h.GetMovieStream)
	r.GET("/api/movies/:id/hls/:segment", h.ProxyHLS)
	r.DELETE("/api/movies/:id", oidc.Middleware(), h.DeleteMovie)
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
	r.Static("/web", "./web")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started on :%s", port)
	r.Run(":" + port)
}
