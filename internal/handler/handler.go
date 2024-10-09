package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/NakarinFIgo/Movies-App/configs"
	"github.com/NakarinFIgo/Movies-App/internal/entities"
	"github.com/NakarinFIgo/Movies-App/pkg/middlewares"
	"github.com/NakarinFIgo/Movies-App/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type Handler struct {
	App configs.Application
}

type Claims struct {
	jwt.RegisteredClaims
}

func (h *Handler) AllMovies(c *fiber.Ctx) error {

	if h.App.DB == nil {
		return utils.ErrorJSON(c, fiber.NewError(fiber.StatusInternalServerError, "database connection is not initialized"))
	}

	movies, err := h.App.DB.AllMovies()
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	return utils.WriteJSON(c, fiber.StatusOK, movies)
}

func (h *Handler) GetMovie(c *fiber.Ctx) error {
	id := c.Params("id") // ใช้ c.Params เพื่อดึงค่า id จาก URL
	movieID, err := strconv.Atoi(id)
	if err != nil {
		return utils.ErrorJSON(c, err) // คืนค่าข้อผิดพลาด
	}

	movie, err := h.App.DB.OneMovie(movieID)
	if err != nil {
		return utils.ErrorJSON(c, err) // คืนค่าข้อผิดพลาด
	}

	return utils.WriteJSON(c, fiber.StatusOK, movie) // ส่งข้อมูลหนังกลับไป
}

func (h *Handler) MovieForEdit(c *fiber.Ctx) error {
	id := c.Params("id") // ใช้ c.Params เพื่อดึงค่า id จาก URL
	movieID, err := strconv.Atoi(id)
	if err != nil {
		return utils.ErrorJSON(c, err) // คืนค่าข้อผิดพลาด
	}

	movie, genres, err := h.App.DB.OneMovieForEdit(movieID)
	if err != nil {
		return utils.ErrorJSON(c, err) // คืนค่าข้อผิดพลาด
	}

	payload := struct {
		Movie  *entities.Movie   `json:"movie"`
		Genres []*entities.Genre `json:"genres"`
	}{
		movie,
		genres,
	}

	return utils.WriteJSON(c, http.StatusOK, payload) // ส่งข้อมูลหนังและประเภทหนังกลับไป
}

func (h *Handler) Authentication(c *fiber.Ctx) error {

	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := utils.ReadJSON(c, &requestPayload)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	user, err := h.App.DB.GetUserByEmail(requestPayload.Email)
	if err != nil {
		return utils.ErrorJSON(c, errors.New("invalid credentials"), fiber.StatusBadRequest)
	}

	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		return utils.ErrorJSON(c, errors.New("invalid credentials"), fiber.StatusBadRequest)
	}
	u := &middlewares.JWTUser{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}

	tokens, err := h.App.Auth.GenerateTokenPair(u)
	if err != nil {
		return utils.ErrorJSON(c, err)

	}

	refreshCookie := h.App.Auth.GetRefreshCookie(tokens.RefreshToken)
	c.Cookie(refreshCookie)

	utils.WriteJSON(c, fiber.StatusOK, tokens)
	return nil
}

func (h *Handler) RefreshToken(c *fiber.Ctx) error {

	// อ่านคุกกี้จาก Fiber context
	cookie := c.Cookies(h.App.Auth.CookieName)
	if cookie == "refresh_token" {
		return utils.ErrorJSON(c, fiber.NewError(fiber.StatusUnauthorized, "unauthorized"))
	}

	claims := &Claims{}
	refreshToken := cookie

	// parse the token to get the claims
	_, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.App.JWTSecret), nil
	})
	if err != nil {
		return utils.ErrorJSON(c, fiber.NewError(fiber.StatusUnauthorized, "unauthorized"))
	}

	// get the user id from the token claims
	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return utils.ErrorJSON(c, fiber.NewError(fiber.StatusUnauthorized, "unknown user"))
	}

	user, err := h.App.DB.GetUserByID(userID)
	if err != nil {
		return utils.ErrorJSON(c, fiber.NewError(fiber.StatusUnauthorized, "unknown user"))
	}

	u := middlewares.JWTUser{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}

	tokenPairs, err := h.App.Auth.GenerateTokenPair(&u)
	if err != nil {
		return utils.ErrorJSON(c, fiber.NewError(fiber.StatusUnauthorized, "error generating tokens"))
	}

	// ใช้ Fiber ในการตั้งค่า cookie
	refreshCookie := h.App.Auth.GetRefreshCookie(tokenPairs.RefreshToken)
	c.Cookie(refreshCookie)

	// ส่ง response กลับในรูปแบบ JSON
	return utils.WriteJSON(c, fiber.StatusOK, tokenPairs)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	expiredCookie := h.App.Auth.GetExpiredRefreshCookie()
	c.Cookie(expiredCookie)
	return c.SendStatus(fiber.StatusAccepted)
}
