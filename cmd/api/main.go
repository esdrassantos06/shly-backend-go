package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/swagger/v2"
	"github.com/redis/go-redis/v9"

	"github.com/esdrassantos06/go-shortener/internal/adapters/handlers"
	"github.com/esdrassantos06/go-shortener/internal/adapters/middleware"
	"github.com/esdrassantos06/go-shortener/internal/adapters/repositories"
	"github.com/esdrassantos06/go-shortener/internal/core/auth"
	"github.com/esdrassantos06/go-shortener/internal/core/services"

	_ "github.com/esdrassantos06/go-shortener/docs"
)

// @title           Zipway URL Shortener API
// @version         1.0.0
// @description     API for URL shortening with support for custom slugs
// @termsOfService  http://swagger.io/terms/

// @host      localhost:8080
// @BasePath  /

// @schemes   http https

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	baseURL := os.Getenv("BASE_URL")

	startTime := time.Now()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	opt, _ := redis.ParseURL(redisURL)
	opt.PoolSize = 50
	opt.MinIdleConns = 20
	opt.MaxRetries = 2
	opt.PoolTimeout = 2 * time.Second
	opt.DialTimeout = 2 * time.Second
	opt.ReadTimeout = 1 * time.Second
	opt.WriteTimeout = 1 * time.Second
	opt.ConnMaxIdleTime = 5 * time.Minute
	opt.ConnMaxLifetime = 30 * time.Minute
	rdb := redis.NewClient(opt)

	linkRepo := repositories.NewPostgresRepo(db)
	cacheRepo := repositories.NewRedisRepo(rdb)
	linkService := services.NewLinkService(linkRepo, cacheRepo)

	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	shortURLDomain := os.Getenv("SHORT_URL_DOMAIN")
	if shortURLDomain == "" {
		shortURLDomain = baseURL
	}
	httpHandler := handlers.NewHTTPHandler(linkService, baseURL, shortURLDomain)

	sessionValidator := auth.NewSessionValidator(db, cacheRepo)
	authMiddleware := middleware.NewAuthMiddleware(sessionValidator)

	app := fiber.New(fiber.Config{
		ServerHeader:      "Zipway",
		AppName:           "Zipway URL Shortener",
		DisableKeepalive:  false,
		ReduceMemoryUsage: true,
	})
	app.Use(logger.New())

	origins := []string{allowedOrigin}
	if allowedOrigin == "" {
		origins = []string{"*"}
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowCredentials: true,
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Cookie"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	app.Get("/swagger/*", swagger.HandlerDefault)

	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":   "Zipway URL Shortener API",
			"version":   "1.0.0",
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"uptime":    time.Since(startTime).String(),
			"swagger":   fmt.Sprintf("%s/swagger", baseURL),
		})
	})

	app.Get("/api/resolve/:slug", httpHandler.ResolveSlug)

	api := app.Group("/api", authMiddleware.RequireAuth)
	api.Post("/shorten", httpHandler.CreateShortLink)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(app.Listen(":" + port))
}
