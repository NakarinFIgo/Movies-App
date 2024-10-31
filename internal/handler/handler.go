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
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	App configs.Application
}

// UserRegisterPayload is the request payload for user login
// swagger:parameters login
type UserLoginPayload struct {
	// Required: true
	// Example: "
	Email string `json:"email"`
	// Required: true
	// Example: "password123"
	Password string `json:"password"`
}

// UserRegisterPayload is the request payload for user registration
// swagger:parameters register
type UserRegisterPayload struct {
	// Required: true
	// Example: "John"
	FirstName string `json:"first_name"`
	// Required: true
	// Example: "Doe"
	LastName string `json:"last_name"`
	// Required: true
	// Example: "john@example.com"
	Email string `json:"email"`
	// Required: true
	// Example: "password123"
	Password string `json:"password"`
}

type Claims struct {
	jwt.RegisteredClaims
}

// login ทำการ login และสร้าง TokenPairs
// @Summary Authentication และสร้าง TokenPairs
// @Description รับข้อมูลอีเมลและรหัสผ่านของผู้ใช้และตรวจสอบความถูกต้อง หลังจากนั้นสร้าง JWT TokenPairs
// @Tags Authentication
// @Accept json
// @Produce json
// @Param requestPayload body UserLoginPayload true "User credentials" example({"email": "string", "password": "string"})
// @Success 202 {object} map[string]interface{} "Token pairs" example({"access_token": "string", "refresh_token": "string"})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error": "Bad Request"})
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error": "Internal Server Error"})
// @Router /api/v1/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {

	var requestPayload UserLoginPayload

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

	// create the response payload (สร้าง payload สำหรับ response)
	responsePayload := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
		} `json:"user"`
	}{
		AccessToken:  tokens.Token, // แก้ไขให้ตรงกับฟิลด์ Token ของคุณ
		RefreshToken: tokens.RefreshToken,
		User: struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
		}{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
		},
	}

	// write the response as JSON (เขียน response เป็น JSON)
	utils.WriteJSON(c, http.StatusAccepted, responsePayload)

	return nil
}

// register เพิ่มผู้ใช้ใหม่ในระบบ
// @Summary เพิ่มผู้ใช้ใหม่
// @Description รับข้อมูลผู้ใช้ใหม่และบันทึกลงในระบบ
// @Tags Authentication
// @Accept json
// @Produce json
// @Param requestPayload body UserRegisterPayload true "User registration data" example({"first_name": "John", "last_name": "Doe", "email": "john@example.com", "password": "password123"})
// @Success 201 {object} map[string]string "message" example({"message": "User created"})
// @Failure 400 {object} map[string]string "Bad Request" example({"error": "Bad Request"})
// @Failure 500 {object} map[string]string "Internal Server Error" example({"error": "Internal Server Error"})
// @Router /api/v1/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var requestPayload UserRegisterPayload

	err := utils.ReadJSON(c, &requestPayload)
	if err != nil {
		utils.ErrorJSON(c, err, http.StatusBadRequest)

	}

	// ตรวจสอบว่าอีเมลนี้มีอยู่แล้วในระบบหรือไม่
	existingUser, _ := h.App.DB.GetUserByEmail(requestPayload.Email)
	if existingUser != nil {
		utils.ErrorJSON(c, errors.New("email already exists"), http.StatusBadRequest)

	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestPayload.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.ErrorJSON(c, err)
	}

	// Create new user
	user := entities.User{
		FirstName: requestPayload.FirstName,
		LastName:  requestPayload.LastName,
		Email:     requestPayload.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Insert user to database
	if _, err := h.App.DB.InsertUser(user); err != nil {
		return utils.ErrorJSON(c, err)
	}
	resp := utils.JSONResponse{
		Error:   false,
		Message: "User created",
	}

	utils.WriteJSON(c, fiber.StatusAccepted, resp)

	return nil
}

// refreshToken รีเฟรชโทเคน JWT
// @Summary รีเฟรชโทเคน JWT
// @Description ตรวจสอบโทเคนที่หมดอายุและสร้างโทเคนใหม่สำหรับผู้ใช้
// @Tags Authentication
// @Produce json
// @Success 200 {object} map[string]string "Token pairs" example({"access_token": "string", "refresh_token": "string"})
// @Failure 401 {object} map[string]string "Unauthorized" example({"error": "Unauthorized"})
// @Failure 500 {object} map[string]string "Internal Server Error" example({"error": "Internal Server Error"})
// @Router /api/v1/refresh [get]
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

// logout ออกจากระบบ
// @Summary ออกจากระบบ
// @Description ลบโทเคนรีเฟรชของผู้ใช้ออกจากระบบ
// @Tags Authentication
// @Produce json
// @Success 202 {object} map[string]string "Accepted" example({"message": "Accepted"})
// @Failure 500 {object} map[string]string "Internal Server Error" example({"error": "Internal Server Error"})
// @Router /api/v1/logout [get]
func (h *Handler) Logout(c *fiber.Ctx) error {
	expiredCookie := h.App.Auth.GetExpiredRefreshCookie()
	c.Cookie(expiredCookie)
	return c.SendStatus(fiber.StatusAccepted)
}

// AllMovies แสดงรายชื่อหนังทั้งหมด
// @Summary แสดงรายชื่อหนังทั้งหมด
// @Description ดึงข้อมูลหนังทั้งหมดจาก database
// @Tags Movies
// @Produce json
// @Success 200 {array} map[string]interface{} "List of all movies" example([{"id":1,"title":"Movie Title","release_date":"2024-08-28","mpaa_rating":"PG","run_time":120,"description":"Description of the movie"}])
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/movies [get]
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

// GetMovie แสดงรายละเอียดของหนังตาม ID
// @Summary แสดงรายละเอียดของหนังตาม ID
// @Description ดึงข้อมูลหนังตาม ID ที่กำหนด
// @Tags Movies
// @Produce json
// @Param id path int true "Movie ID"
// @Success 200 {object} map[string]interface{} "Movie details" example({"id":1,"title":"Movie Title","release_date":"2024-08-28","mpaa_rating":"PG","run_time":120,"description":"Description of the movie"})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error":"Invalid ID"})
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/movies/{id} [get]
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

// MovieForEdit ดึงข้อมูลหนังและประเภทหนังสำหรับการแก้ไข
// @Summary ดึงข้อมูลหนังและประเภทหนังสำหรับการแก้ไข
// @Description ดึงข้อมูลหนังและประเภทหนังสำหรับการแก้ไขตาม ID
// @Tags Movies
// @Produce json
// @Security BearerAuth
// @Param id path int true "Movie ID"
// @Success 200 {object} map[string]interface{} "Movie and genres details" example({"movie":{"id":1,"title":"Movie Title"},"genres":[{"id":1,"name":"Genre Name"}]})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error":"Invalid ID"})
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/admin/movies/{id} [get]
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

// MovieCatalog แสดงรายชื่อหนังในแคตตาล็อก
// @Summary แสดงรายชื่อหนังในแคตตาล็อก
// @Description ดึงข้อมูลหนังทั้งหมดจากแคตตาล็อก
// @Tags Movies
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{} "List of movies in catalog" example([{"id":1,"title":"Catalog Movie Title","release_date":"2024-08-28","mpaa_rating":"PG","run_time":90,"description":"Description of the catalog movie"}])
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/admin/movies [get]
func (h *Handler) MovieCatalog(c *fiber.Ctx) error {
	movies, err := h.App.DB.AllMovies()
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	return utils.WriteJSON(c, fiber.StatusOK, movies)
}

// AllGenres แสดงประเภทหนังทั้งหมด
// @Summary แสดงประเภทหนังทั้งหมด
// @Description ดึงข้อมูลประเภทหนังทั้งหมด
// @Tags Genres
// @Produce json
// @Success 200 {array} map[string]interface{} "List of all genres" example([{"id":1,"name":"Action"},{"id":2,"name":"Drama"}])
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/genres [get]
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

// InsertMovie เพิ่มหนังใหม่
// @Summary เพิ่มหนังใหม่
// @Description เพิ่มหนังใหม่ไปยังฐานข้อมูล
// @Tags Movies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param movie body object true "Movie data" example({"title":"New Movie","release_date":"2024-08-28","mpaa_rating":"PG","run_time":120,"description":"New movie description"})
// @Success 202 {object} map[string]interface{} "Movie created" example({"message":"movie updated"})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error":"Invalid data"})
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/admin/movies [post]
func (h *Handler) InsertMovie(c *fiber.Ctx) error {
	var movie entities.Movie

	err := utils.ReadJSON(c, &movie)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	movie = h.GetPoster(movie)

	movie.CreatedAt = time.Now()
	movie.UpdatedAt = time.Now()

	newID, err := h.App.DB.InsertMovie(movie)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	err = h.App.DB.UpdateMovieGenres(newID, movie.GenresArray)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	err = h.App.DB.UpdateMovieGenres(newID, movie.GenresArray)
	if err != nil {
		return utils.ErrorJSON(c, err)
	}

	resp := utils.JSONResponse{
		Error:   false,
		Message: "movie updated",
	}

	utils.WriteJSON(c, fiber.StatusAccepted, resp)

	return nil
}

// UpdateMovie แก้ไขข้อมูลหนัง
// @Summary แก้ไขข้อมูลหนัง
// @Description แก้ไขข้อมูลหนังตาม ID ที่กำหนด
// @Tags Movies
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param movie body object true "Updated movie data" example({"id":1,"title":"Updated Movie Title","release_date":"2024-08-28","mpaa_rating":"PG","run_time":130,"description":"Updated movie description"})
// @Success 202 {object} map[string]interface{} "Movie updated" example({"message":"movie updated"})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error":"Invalid data"})
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/admin/movies/{id} [put]
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

// DeleteMovie ลบหนังตาม ID
// @Summary ลบหนังตาม ID
// @Description ลบข้อมูลหนังตาม ID ที่กำหนด
// @Tags Movies
// @Produce json
// @Security BearerAuth
// @Param id path int true "Movie ID"
// @Success 202 {object} map[string]interface{} "Movie deleted" example({"message":"movie deleted"})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error":"Invalid ID"})
// @Failure 500 {object} map[string]interface{} "Internal Server Error" example({"error":"Internal Server Error"})
// @Router /api/v1/admin/movies/{id} [delete]
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
