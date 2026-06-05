package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"titip-jejak-api/internal/middleware"
	"testing"
	"time"

	"titip-jejak-api/internal/handler"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/model"
	"titip-jejak-api/internal/repository"
	"titip-jejak-api/internal/usecase"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Globals ────────────────────────────────────────────────────────────────

var (
	testDB     *gorm.DB
	testRouter *gin.Engine
)

// ── Setup ──────────────────────────────────────────────────────────────────

func setupTestDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME_TEST"), // pastikan ada DB_NAME_TEST di .env
		os.Getenv("DB_PORT"),
		os.Getenv("DB_TIMEZONE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`).Error; err != nil {
		return nil, err
	}

	enums := []string{
		`DO $$ BEGIN
            CREATE TYPE report_type AS ENUM ('missing', 'found');
        EXCEPTION WHEN duplicate_object THEN NULL;
        END $$;`,

		`DO $$ BEGIN
            CREATE TYPE report_gender AS ENUM ('male', 'female', 'unknown');
        EXCEPTION WHEN duplicate_object THEN NULL;
        END $$;`,

		`DO $$ BEGIN
            CREATE TYPE report_status AS ENUM ('active', 'resolved');
        EXCEPTION WHEN duplicate_object THEN NULL;
        END $$;`,
	}

	for _, enum := range enums {
		if err := db.Exec(enum).Error; err != nil {
			return nil, err
		}
	}

	if err := db.AutoMigrate(&model.User{}, &model.Report{}, &model.Match{}, &model.Notification{}); err != nil {
		return nil, err
	}

	return db, nil
}

func setupRouter(db *gorm.DB) *gin.Engine {
	validate := validator.New()

	userRepo := repository.NewUserRepository(db)
	uc := usecase.NewUserUsecase(userRepo, validate)
	h := handler.NewUserHandlerImpl(uc)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.ErrorRecovery())

	api := r.Group("/api/v1/auth")
	api.POST("/register", h.Create)
	api.POST("/login", h.Login)
	api.POST("/refresh", h.RefreshToken)

	authorized := api.Group("/")
	authorized.Use(middleware.AuthMiddleware())
	{
		authorized.POST("/me", h.Me)
		authorized.GET("/logout", h.Logout)
	}

	return r
}

func truncateTables(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
}

func TestMain(m *testing.M) {
	if err := godotenv.Load("../.env.test"); err != nil {
		panic("Failed to load .env.test: " + err.Error())
	}

	db, err := setupTestDB()
	if err != nil {
		panic(err)
	}
	testDB = db
	testRouter = setupRouter(testDB)

	code := m.Run()
	os.Exit(code)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func doPost(router *gin.Engine, url, body string, headers ...map[string]string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func doPostWithCookie(router *gin.Engine, url, body string, cookies []*http.Cookie, headers ...map[string]string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func doGetWithToken(router *gin.Engine, url, token string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func doGetWithCookie(router *gin.Engine, url string, cookies []*http.Cookie, headers ...map[string]string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func parseBody(r *http.Response) map[string]any {
	b, _ := io.ReadAll(r.Body)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}

// seedUser mendaftarkan user dan mengembalikan responsenya
func seedUser(email, name, role string) {
	doPost(testRouter, "/api/v1/auth/register", fmt.Sprintf(`{
		"name":     "%s",
		"email":    "%s",
		"password": "rahasia123",
		"role":     "%s"
	}`, name, email, role))
}

// ── Register (Create User) ─────────────────────────────────────────────────

// 1. Happy path — semua field valid termasuk phone (opsional)
func TestRegisterSuccess(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Budi Santoso",
		"email":    "budi@mail.com",
		"password": "rahasia123",
		"role":     "finder"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "User created successfully", body["message"])

	data := body["data"].(map[string]any)
	assert.Equal(t, "Budi Santoso", data["name"])
	assert.Equal(t, "budi@mail.com", data["email"])
	assert.Equal(t, "finder", data["role"])
	assert.NotEmpty(t, data["id"])
	assert.Nil(t, data["password"]) // password tidak boleh muncul di response
}

// 2. Happy path — dengan phone opsional
func TestRegisterSuccessWithPhone(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Ani Rahayu",
		"email":    "ani@mail.com",
		"password": "rahasia123",
		"role":     "seeker",
		"phone":    "08123456789"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, "Ani Rahayu", data["name"])
	assert.Equal(t, "seeker", data["role"])
}

// 3. Happy path — role volunteer
func TestRegisterSuccessRoleVolunteer(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Candra",
		"email":    "candra@mail.com",
		"password": "rahasia123",
		"role":     "volunteer"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, "volunteer", data["role"])
}

// 4. Password tidak boleh ada di response
func TestRegisterPasswordNotExposed(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Dian",
		"email":    "dian@mail.com",
		"password": "rahasia123",
		"role":     "finder"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 201, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Nil(t, data["password"])
}

// 5. Duplicate email → 409 CONFLICT
func TestRegisterDuplicateEmail(t *testing.T) {
	truncateTables(testDB)

	payload := `{
		"name":     "Eko",
		"email":    "eko@mail.com",
		"password": "rahasia123",
		"role":     "finder"
	}`

	first := doPost(testRouter, "/api/v1/auth/register", payload)
	assert.Equal(t, 201, first.StatusCode)

	second := doPost(testRouter, "/api/v1/auth/register", payload)
	body := parseBody(second)

	assert.Equal(t, 409, second.StatusCode)
	assert.Equal(t, "CONFLICT", body["status"])
	assert.NotEmpty(t, body["error"])
}

// 6. Field name kosong → 400
func TestRegisterMissingName(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"email":    "noname@mail.com",
		"password": "rahasia123",
		"role":     "finder"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 7. Field email kosong → 400
func TestRegisterMissingEmail(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Fajar",
		"password": "rahasia123",
		"role":     "finder"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 8. Field password kosong → 400
func TestRegisterMissingPassword(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":  "Gilang",
		"email": "gilang@mail.com",
		"role":  "finder"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 9. Field role kosong → 400
func TestRegisterMissingRole(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Hendra",
		"email":    "hendra@mail.com",
		"password": "rahasia123"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 10. Format email tidak valid → 400
func TestRegisterInvalidEmailFormat(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Indra",
		"email":    "bukan-email",
		"password": "rahasia123",
		"role":     "finder"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 11. Role tidak valid (bukan finder/seeker/volunteer) → 400
func TestRegisterInvalidRole(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Joko",
		"email":    "joko@mail.com",
		"password": "rahasia123",
		"role":     "admin"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 12. Body kosong {} → 400
func TestRegisterEmptyBody(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 13. JSON tidak valid → 400
func TestRegisterInvalidJSON(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{ invalid json }`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 14. Verifikasi data tersimpan di DB dengan benar
func TestRegisterDataSavedToDB(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Kiki",
		"email":    "kiki@mail.com",
		"password": "rahasia123",
		"role":     "seeker"
	}`)
	assert.Equal(t, 201, resp.StatusCode)

	var user model.User
	testDB.Where("email = ?", "kiki@mail.com").First(&user)

	assert.Equal(t, "Kiki", user.Name)
	assert.Equal(t, "kiki@mail.com", user.Email)
	assert.Equal(t, model.RoleSeeker, user.Role)
	assert.NotEmpty(t, user.ID)
	// Password harus di-hash, bukan plain text
	assert.NotEqual(t, "rahasia123", user.Password)
	assert.True(t, helper.CheckPasswordHash("rahasia123", user.Password))
}

// 15. Verifikasi phone tersimpan di DB jika dikirim
func TestRegisterPhoneSavedToDB(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/register", `{
		"name":     "Lina",
		"email":    "lina@mail.com",
		"password": "rahasia123",
		"role":     "volunteer",
		"phone":    "08129999888"
	}`)
	assert.Equal(t, 201, resp.StatusCode)

	var user model.User
	testDB.Where("email = ?", "lina@mail.com").First(&user)

	assert.NotNil(t, user.Phone)
	assert.Equal(t, "08129999888", *user.Phone)
}

func loginMobile(email string) (accessToken, refreshToken string) {
	resp := doPost(testRouter, "/api/v1/auth/login", fmt.Sprintf(`{
		"email":    "%s",
		"password": "rahasia123"
	}`, email))
	body := parseBody(resp)
	data, _ := body["data"].(map[string]any)
	tokens, _ := data["tokens"].(map[string]any)
	accessToken, _ = tokens["access_token"].(string)
	refreshToken, _ = tokens["refresh_token"].(string)
	return
}

// loginWeb melakukan login sebagai web client dan mengembalikan cookies dari response
func loginWeb(email string) []*http.Cookie {
	resp := doPost(testRouter, "/api/v1/auth/login", fmt.Sprintf(`{
		"email":    "%s",
		"password": "rahasia123"
	}`, email), map[string]string{"X-Client-Type": "web"})
	return resp.Cookies()
}

// ── LOGIN ──────────────────────────────────────────────────────────────────

// 1. Mobile - login berhasil → 200, token di body
func TestLoginMobileSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("budi@mail.com", "Budi", "finder")

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email":    "budi@mail.com",
		"password": "rahasia123"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Login successful", body["message"])

	data := body["data"].(map[string]any)
	tokens := data["tokens"].(map[string]any)
	assert.NotEmpty(t, tokens["access_token"])
	assert.NotEmpty(t, tokens["refresh_token"])

	user := data["user"].(map[string]any)
	assert.Equal(t, "budi@mail.com", user["email"])
	assert.Nil(t, user["password"]) // password tidak boleh ada di response
}

// 2. Mobile - token di body berupa string valid (bukan kosong)
func TestLoginMobileTokenNotEmpty(t *testing.T) {
	truncateTables(testDB)
	seedUser("ani@mail.com", "Ani", "seeker")

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email":    "ani@mail.com",
		"password": "rahasia123"
	}`)
	body := parseBody(resp)

	data := body["data"].(map[string]any)
	tokens := data["tokens"].(map[string]any)
	assert.Greater(t, len(tokens["access_token"].(string)), 10)
	assert.Greater(t, len(tokens["refresh_token"].(string)), 10)
}

// 3. Web - login berhasil → 200, token di cookie bukan di body
func TestLoginWebSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("candra@mail.com", "Candra", "volunteer")

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email":    "candra@mail.com",
		"password": "rahasia123"
	}`, map[string]string{"X-Client-Type": "web"})
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	// Token tidak boleh ada di body
	data := body["data"].(map[string]any)
	assert.Nil(t, data["tokens"])

	// Token harus ada di cookie
	var hasAccessToken, hasRefreshToken bool
	for _, c := range resp.Cookies() {
		if c.Name == "access_token" && c.Value != "" {
			hasAccessToken = true
		}
		if c.Name == "refresh_token" && c.Value != "" {
			hasRefreshToken = true
		}
	}
	assert.True(t, hasAccessToken, "access_token cookie harus ada")
	assert.True(t, hasRefreshToken, "refresh_token cookie harus ada")
}

// 4. Web - cookie harus HttpOnly
func TestLoginWebCookieIsHttpOnly(t *testing.T) {
	truncateTables(testDB)
	seedUser("dian@mail.com", "Dian", "finder")

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email":    "dian@mail.com",
		"password": "rahasia123"
	}`, map[string]string{"X-Client-Type": "web"})

	for _, c := range resp.Cookies() {
		if c.Name == "access_token" || c.Name == "refresh_token" {
			assert.True(t, c.HttpOnly, "cookie %s harus HttpOnly", c.Name)
		}
	}
}

// 5. Email tidak terdaftar → 404
func TestLoginEmailNotFound(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email":    "tidakada@mail.com",
		"password": "rahasia123"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 6. Password salah → 401
func TestLoginWrongPassword(t *testing.T) {
	truncateTables(testDB)
	seedUser("eko@mail.com", "Eko", "finder")

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email":    "eko@mail.com",
		"password": "passwordsalah"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 7. Email kosong → 400
func TestLoginMissingEmail(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"password": "rahasia123"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 8. Password kosong → 400
func TestLoginMissingPassword(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/login", `{
		"email": "fajar@mail.com"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 9. Body kosong → 400
func TestLoginEmptyBody(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/login", `{}`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 10. JSON tidak valid → 400
func TestLoginInvalidJSON(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/login", `{ invalid json }`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// ── REFRESH TOKEN ──────────────────────────────────────────────────────────

// 11. Mobile - refresh berhasil → 200, access token baru di body
func TestRefreshTokenMobileSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("gilang@mail.com", "Gilang", "finder")
	_, refreshToken := loginMobile("gilang@mail.com")

	resp := doPost(testRouter, "/api/v1/auth/refresh", fmt.Sprintf(`{
		"refresh_token": "%s"
	}`, refreshToken))
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Token refreshed successfully", body["message"])

	data := body["data"].(map[string]any)
	assert.NotEmpty(t, data["access_token"])
}

// 12. Mobile - access token baru berbeda dari sebelumnya
func TestRefreshTokenMobileNewTokenDifferent(t *testing.T) {
	truncateTables(testDB)
	seedUser("hendra@mail.com", "Hendra", "seeker")
	oldAccess, refreshToken := loginMobile("hendra@mail.com")

	time.Sleep(1 * time.Second)

	resp := doPost(testRouter, "/api/v1/auth/refresh", fmt.Sprintf(`{
		"refresh_token": "%s"
	}`, refreshToken))
	body := parseBody(resp)

	data := body["data"].(map[string]any)
	newAccess, _ := data["access_token"].(string)
	assert.NotEqual(t, oldAccess, newAccess, "access token baru harus berbeda")
}

// 13. Web - refresh berhasil → 200, access token baru di cookie
func TestRefreshTokenWebSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("indra@mail.com", "Indra", "volunteer")
	cookies := loginWeb("indra@mail.com")

	resp := doPostWithCookie(testRouter, "/api/v1/auth/refresh", `{}`, cookies,
		map[string]string{"X-Client-Type": "web"})
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	// Token tidak boleh ada di body
	assert.Nil(t, body["data"])

	// access_token baru harus ada di cookie
	var hasNewAccessToken bool
	for _, c := range resp.Cookies() {
		if c.Name == "access_token" && c.Value != "" {
			hasNewAccessToken = true
		}
	}
	assert.True(t, hasNewAccessToken, "access_token cookie baru harus ada")
}

// 14. Mobile - refresh token tidak dikirim → 401
func TestRefreshTokenMobileMissingToken(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/refresh", `{
		"refresh_token": ""
	}`)
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 15. Mobile - refresh token tidak valid → 401
func TestRefreshTokenMobileInvalidToken(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/refresh", `{
		"refresh_token": "ini.bukan.token.valid"
	}`)
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 16. Web - cookie refresh_token tidak ada → 401
func TestRefreshTokenWebMissingCookie(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/refresh", `{}`,
		map[string]string{"X-Client-Type": "web"})
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 17. Mobile - body kosong → 400
func TestRefreshTokenMobileEmptyBody(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/refresh", `{}`)
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 18. Mobile - JSON tidak valid → 400
func TestRefreshTokenInvalidJSON(t *testing.T) {
	truncateTables(testDB)

	resp := doPost(testRouter, "/api/v1/auth/refresh", `{ invalid json }`)
	body := parseBody(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// ── ME ─────────────────────────────────────────────────────────────────────

// 19. Mobile - me berhasil dengan Bearer token → 200
func TestMeMobileSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("joko@mail.com", "Joko", "finder")
	accessToken, _ := loginMobile("joko@mail.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, "joko@mail.com", data["email"])
	assert.Equal(t, "Joko", data["name"])
	assert.Equal(t, "finder", data["role"])
	assert.Nil(t, data["password"])
}

// 20. Web - me berhasil dengan cookie → 200
func TestMeWebSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("kiki@mail.com", "Kiki", "seeker")
	cookies := loginWeb("kiki@mail.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	req.Header.Set("X-Client-Type", "web")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, "kiki@mail.com", data["email"])
	assert.Equal(t, "Kiki", data["name"])
}

// 21. Me tanpa token → 401
func TestMeNoToken(t *testing.T) {
	truncateTables(testDB)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 22. Me dengan token tidak valid → 401
func TestMeInvalidToken(t *testing.T) {
	truncateTables(testDB)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer ini.bukan.token.valid")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 23. Me dengan Bearer prefix salah → 401
func TestMeInvalidBearerFormat(t *testing.T) {
	truncateTables(testDB)
	seedUser("lina@mail.com", "Lina", "volunteer")
	accessToken, _ := loginMobile("lina@mail.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Token "+accessToken) // bukan "Bearer"
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 24. Me - data yang dikembalikan sesuai user yang login (bukan user lain)
func TestMeReturnCorrectUser(t *testing.T) {
	truncateTables(testDB)
	seedUser("mira@mail.com", "Mira", "finder")
	seedUser("nanda@mail.com", "Nanda", "seeker")
	accessToken, _ := loginMobile("mira@mail.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, "mira@mail.com", data["email"])
	assert.NotEqual(t, "nanda@mail.com", data["email"])
}

// ── LOGOUT ─────────────────────────────────────────────────────────────────

// 25. Mobile - logout berhasil dengan Bearer token → 200
func TestLogoutMobileSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("oki@mail.com", "Oki", "finder")
	accessToken, _ := loginMobile("oki@mail.com")

	resp := doGetWithToken(testRouter, "/api/v1/auth/logout", accessToken)
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Logout successful", body["message"])
}

// 26. Web - logout berhasil → 200, cookie di-clear
func TestLogoutWebSuccess(t *testing.T) {
	truncateTables(testDB)
	seedUser("pita@mail.com", "Pita", "seeker")
	cookies := loginWeb("pita@mail.com")

	resp := doGetWithCookie(testRouter, "/api/v1/auth/logout", cookies,
		map[string]string{"X-Client-Type": "web"})
	body := parseBody(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Logout successful", body["message"])
}

// 27. Web - setelah logout cookie access_token dan refresh_token harus di-clear (MaxAge <= 0 atau value kosong)
func TestLogoutWebCookieCleared(t *testing.T) {
	truncateTables(testDB)
	seedUser("rafi@mail.com", "Rafi", "volunteer")
	cookies := loginWeb("rafi@mail.com")

	resp := doGetWithCookie(testRouter, "/api/v1/auth/logout", cookies,
		map[string]string{"X-Client-Type": "web"})

	for _, c := range resp.Cookies() {
		if c.Name == "access_token" || c.Name == "refresh_token" {
			assert.True(t, c.MaxAge <= 0 || c.Value == "",
				"cookie %s harus di-clear setelah logout", c.Name)
		}
	}
}

// 28. Logout tanpa token → 401
func TestLogoutNoToken(t *testing.T) {
	truncateTables(testDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 29. Logout dengan token tidak valid → 401
func TestLogoutInvalidToken(t *testing.T) {
	truncateTables(testDB)

	resp := doGetWithToken(testRouter, "/api/v1/auth/logout", "ini.bukan.token.valid")
	body := parseBody(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 30. Mobile - setelah logout, token lama masih valid (stateless — tidak ada blacklist)
// Test ini memastikan bahwa server tidak menyimpan state token yang logout
func TestLogoutMobileTokenStillValidStateless(t *testing.T) {
	truncateTables(testDB)
	seedUser("sari@mail.com", "Sari", "finder")
	accessToken, _ := loginMobile("sari@mail.com")

	// Logout
	doGetWithToken(testRouter, "/api/v1/auth/logout", accessToken)

	// Token lama masih bisa dipakai (stateless) — me masih 200
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	resp := rec.Result()

	assert.Equal(t, 200, resp.StatusCode,
		"server stateless — token lama masih valid setelah logout mobile")
}
