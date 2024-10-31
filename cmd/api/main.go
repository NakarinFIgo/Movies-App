package main

import (
	"log"
	"os"
	"time"

	"github.com/NakarinFIgo/Movies-App/configs"
	_ "github.com/NakarinFIgo/Movies-App/docs"
	"github.com/NakarinFIgo/Movies-App/internal/handler"
	"github.com/NakarinFIgo/Movies-App/internal/repository"
	"github.com/NakarinFIgo/Movies-App/pkg/db"
	"github.com/NakarinFIgo/Movies-App/pkg/middlewares"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"
)

// @title Movies API with GO and PostgreSQL
// @version 1.0
// @description This is a Movies API with GO and PostgreSQL
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	var cfx configs.Application

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfx.JWTSecret = os.Getenv("JWT_SECRET")
	cfx.JWTIssuer = os.Getenv("JWT_ISSUER")
	cfx.JWTAudience = os.Getenv("JWT_AUDIENCE")
	cfx.CookieDomain = os.Getenv("COOKIE_DOMAIN")
	cfx.Domain = os.Getenv("DOMAIN")
	cfx.APIKey = os.Getenv("API_KEY")

	databaseRepo := db.DBConnection()
	if databaseRepo == nil {
		log.Fatal("Failed to connect to the database")
	}

	moviesRepo := &repository.PostgresRepository{DB: databaseRepo}
	cfx.DB = moviesRepo

	cfx.Auth = middlewares.Auth{
		Issuer:        cfx.JWTIssuer,
		Audience:      cfx.JWTAudience,
		Secret:        cfx.JWTSecret,
		TokenExpiry:   time.Minute * 15,
		RefreshExpiry: time.Hour * 24 * 7,
		CookieDomain:  "localhost",
		CookiePath:    "/",
		CookieName:    "refresh_token",
	}

	app := fiber.New()

	h := &handler.Handler{
		App: cfx,
	}

	app.Use(middlewares.Enablecors())
	app.Get("/swagger/*", swagger.HandlerDefault)

	// API Routes
	app.Route("/api/v1", func(router fiber.Router) {
		router.Post("/login", h.Login)
		router.Get("/refresh", h.RefreshToken)
		router.Get("/register", h.Register)
		router.Get("/logout", h.Logout)

		router.Get("/movies", h.AllMovies)
		router.Get("/movies/:id", h.GetMovie)
		router.Get("/genres", h.AllGenres)

		// Admin routes with JWT middleware
		admin := router.Group("/admin")
		admin.Use(middlewares.JwtMiddleware())
		admin.Get("/movies", h.MovieCatalog)
		admin.Get("/movies/:id", h.MovieForEdit)
		admin.Post("/movies", h.InsertMovie)
		admin.Put("/movies/:id", h.UpdateMovie)
		admin.Delete("/movies/:id", h.DeleteMovie)
	})

	err = app.Listen(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
