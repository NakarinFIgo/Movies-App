package shared

import "github.com/gofiber/fiber/v2"

type Application struct {
	Auth AuthInterface // Example of an interface
}

type AuthInterface interface {
	GetTokenFromHeaderAndVerify(c *fiber.Ctx) (string, *Claims, error)
}
