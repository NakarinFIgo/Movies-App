package main

import (
	"log"
	"os"
	"time"

	"github.com/NakarinFIgo/Movies-App/configs"
	"github.com/NakarinFIgo/Movies-App/internal/handler"
	"github.com/NakarinFIgo/Movies-App/internal/repository"
	"github.com/NakarinFIgo/Movies-App/pkg/db"
	"github.com/NakarinFIgo/Movies-App/pkg/middlewares"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

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

	app.Post("/authenticate", h.Authentication)
	app.Get("/refresh", h.RefreshToken)
	app.Get("/logout", h.Logout)

	app.Get("/movies", h.AllMovies)
	app.Get("/movies/:id", h.GetMovie)
	app.Get("genres", h.AllGenres)

	app.Route("/admin", func(router fiber.Router) {
		router.Use(h.App.Auth.AuthRequired())

		router.Get("/movies", h.MovieCatalog)
		router.Get("/movies/:id", h.MovieForEdit)
		router.Post("/movies", h.InsertMovie)
		router.Put("/movies/:id", h.UpdateMovie)
		router.Delete("/movies/:id", h.DeleteMovie)

	})

	err = app.Listen(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
