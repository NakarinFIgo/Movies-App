package configs

import (
	"github.com/NakarinFIgo/Movies-App/internal/repository"
	"github.com/NakarinFIgo/Movies-App/pkg/middlewares"
)

type Application struct {
	DB           repository.DatabaseRepo
	DSN          string
	Domain       string
	Auth         middlewares.Auth
	JWTSecret    string
	JWTIssuer    string
	JWTAudience  string
	CookieDomain string
	APIKey       string
}
