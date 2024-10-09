package utils

import "github.com/gofiber/fiber/v2"

type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func WriteJSON(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(data)
}

func ReadJSON(c *fiber.Ctx, data interface{}) error {
	if err := c.BodyParser(data); err != nil {
		return err
	}
	return nil
}

func ErrorJSON(c *fiber.Ctx, err error, status ...int) error {
	statusCode := fiber.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return WriteJSON(c, statusCode, payload)
}
