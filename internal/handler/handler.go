package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

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

func (h *Handler) MovieCatalog(c *fiber.Ctx) error {
	movies, err := h.App.DB.AllMovies()
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	return utils.WriteJSON(c, fiber.StatusOK, movies)
}
func (h *Handler) AllGenres(c *fiber.Ctx) error {
	genres, err := h.App.DB.AllGenres()
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	_ = utils.WriteJSON(c, fiber.StatusOK, genres)

	return nil
}

func (h *Handler) GetPoster(movie entities.Movie) entities.Movie {
	type TheMovieDB struct {
		Page    int `json:"page"`
		Results []struct {
			PosterPath string `json:"poster_path"`
		} `json:"results"`
		TotalPages int `json:"total_pages"`
	}

	theUrl := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?api_key=%s", h.App.APIKey)
	queryUrl := theUrl + "&query=" + url.QueryEscape(movie.Title)

	req, err := http.NewRequest("GET", queryUrl, nil)
	if err != nil {
		log.Println(err)
		return movie
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return movie
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return movie
	}

	var responseObject TheMovieDB
	if err := json.Unmarshal(bodyBytes, &responseObject); err != nil {
		log.Println(err)
		return movie
	}

	if len(responseObject.Results) > 0 {
		movie.Image = responseObject.Results[0].PosterPath
	}

	return movie
}

func (h *Handler) InsertMovie(c *fiber.Ctx) error {
	var movie entities.Movie

	err := utils.ReadJSON(c, &movie)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	movie = h.GetPoster(movie)

	movie.CreatedAt = time.Now()
	movie.UpdatedAt = time.Now()

	newID, err := h.App.DB.InsertMovie(movie)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	err = h.App.DB.UpdateMovieGenres(newID, movie.GenresArray)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	err = h.App.DB.UpdateMovieGenres(newID, movie.GenresArray)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	resp := utils.JSONResponse{
		Error:   false,
		Message: "movie updated",
	}

	utils.WriteJSON(c, fiber.StatusAccepted, resp)

	return nil
}

func (h *Handler) UpdateMovie(c *fiber.Ctx) error {
	var payload entities.Movie

	err := utils.ReadJSON(c, &payload)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	movie, err := h.App.DB.OneMovie(payload.ID)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	movie.Title = payload.Title
	movie.ReleaseDate = payload.ReleaseDate
	movie.Description = payload.Description
	movie.MPAARating = payload.MPAARating
	movie.RunTime = payload.RunTime
	movie.UpdatedAt = time.Now()

	err = h.App.DB.UpdateMovie(*movie)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	err = h.App.DB.UpdateMovieGenres(movie.ID, payload.GenresArray)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	resp := utils.JSONResponse{
		Error:   false,
		Message: "movie updated",
	}

	utils.WriteJSON(c, fiber.StatusAccepted, resp)

	return nil
}

func (h *Handler) DeleteMovie(c *fiber.Ctx) error {
	id := c.Params("id")
	movieID, err := strconv.Atoi(id)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	err = h.App.DB.DeleteMovie(movieID)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	resp := utils.JSONResponse{
		Error:   false,
		Message: "movie deleted",
	}

	utils.WriteJSON(c, http.StatusAccepted, resp)

	return nil
}
