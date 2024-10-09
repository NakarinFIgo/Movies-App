package main

import (
	"flag"
	"log"
	"time"

	"github.com/NakarinFIgo/Movies-App/configs"
	"github.com/NakarinFIgo/Movies-App/internal/handler"
	"github.com/NakarinFIgo/Movies-App/internal/repository"
	"github.com/NakarinFIgo/Movies-App/pkg/db"
	"github.com/NakarinFIgo/Movies-App/pkg/middlewares"
	"github.com/gofiber/fiber/v2"
)

func main() {

	var cfx configs.Application

	flag.StringVar(&cfx.DSN, "dsn", "host=localhost port=5432 dbname=gosampledb user=postgres password=123456 sslmode=disable timezone=UTC connect_timeout=5", "Postgres connection string")

	flag.StringVar(&cfx.JWTSecret, "jwt-secret", "verysecret", "signing secret")
	flag.StringVar(&cfx.JWTIssuer, "jwt-issuer", "example.com", "signing issuer")
	flag.StringVar(&cfx.JWTAudience, "jwt-audience", "example.com", "signing audience")
	flag.StringVar(&cfx.CookieDomain, "cookie-domain", "localhost", "cookie domain")
	flag.StringVar(&cfx.Domain, "domain", "example.com", "domain")
	flag.StringVar(&cfx.APIKey, "api-key", "b41447e6319d1cd467306735632ba733", "api key")

	flag.Parse()

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

	app.Route("/admin", func(router fiber.Router) {
		router.Use(h.App.Auth.AuthRequired())
		router.Get("/movies/:id", h.MovieForEdit)
	})

	err := app.Listen(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
