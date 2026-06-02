package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/rohilprajapati/shortern_url_redis_fiber/routes"
)

// Fixed: Changed '**fiber.App' to a single pointer '*fiber.App'
func setupRoutes(app *fiber.App) {
	// Fixed: Changed completely uppercase .GET/.POST to Fiber v2 standard .Get/.Post
	app.Get("/:url", routes.ResolveURL)
	app.Post("/api/v1", routes.ShortenURL)
}

func main() {
	// Attempt to load .env, but don't treat a failure as a warning
	// if critical environment variables are already present.
	_ = godotenv.Load()

	app := fiber.New()
	app.Use(logger.New())

	setupRoutes(app)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000" // Local manual run fallback
	}

	if port[0] != ':' {
		port = ":" + port
	}

	fmt.Printf("Server is starting on port %s...\n", port)
	log.Fatal(app.Listen(port))
}
