package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"titip-jejak-api/internal/handler"
	"titip-jejak-api/internal/middleware"
	"titip-jejak-api/internal/model"
	"titip-jejak-api/internal/repository"
	"titip-jejak-api/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ── Setup ──────────────────────────────────────────────────────────────────

func setupReportRouter(db *gorm.DB) *gin.Engine {
	validate := validator.New()

	// User layer (dibutuhkan untuk auth middleware & seed)
	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, validate)
	userHandler := handler.NewUserHandlerImpl(userUsecase)

	// Report layer — skip cloudinary dengan cld = nil,
	// test upload foto pakai mock terpisah di bawah
	reportRepo := repository.NewReportRepository(db)
	reportUsecase := usecase.NewReportUsecase(reportRepo, validate, nil)
	reportHandler := handler.NewReportHandlerImpl(reportUsecase)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.ErrorRecovery())

	api := r.Group("/api/v1")

	// Auth (dibutuhkan untuk login & seed token)
	auth := api.Group("/auth")
	auth.POST("/register", userHandler.Create)
	auth.POST("/login", userHandler.Login)

	// Reports
	reports := api.Group("/reports")
	reports.GET("", reportHandler.GetAll)
	reports.GET("/:id", reportHandler.GetByID)

	reportsPrivate := reports.Group("")
	reportsPrivate.Use(middleware.AuthMiddleware())
	reportsPrivate.GET("/my", reportHandler.GetMyReports)
	reportsPrivate.POST("", reportHandler.Create)
	reportsPrivate.PUT("/:id", reportHandler.Update)
	reportsPrivate.DELETE("/:id", reportHandler.Delete)
	reportsPrivate.POST("/:id/photo", reportHandler.UploadPhoto)

	// Map
	api.GET("/map/pins", reportHandler.GetMapPins)

	return r
}

func truncateReports(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE reports RESTART IDENTITY CASCADE")
}

// ── HTTP helpers (report-specific) ────────────────────────────────────────

func doPut(router *gin.Engine, url, body, token string) *http.Response {
	req := httptest.NewRequest(http.MethodPut, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func doDelete(router *gin.Engine, url, token string) *http.Response {
	req := httptest.NewRequest(http.MethodDelete, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func doGet(router *gin.Engine, url string, token ...string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func doPostAuth(router *gin.Engine, url, body, token string) *http.Response {
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

func parseBodyReport(r *http.Response) map[string]any {
	b, _ := io.ReadAll(r.Body)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}

// ── Seed helpers ──────────────────────────────────────────────────────────

var reportRouter *gin.Engine

func init() {
	// reportRouter di-init saat TestMain sudah setup testDB
	// Kita override di TestMain atau gunakan lazy init via getReportRouter()
}

func getReportRouter() *gin.Engine {
	if reportRouter == nil {
		reportRouter = setupReportRouter(testDB)
	}
	return reportRouter
}

// registerAndLogin mendaftarkan user baru dan mengembalikan access token
func registerAndLogin(email, name, role string) string {
	router := getReportRouter()
	doPostAuth(router, "/api/v1/auth/register", fmt.Sprintf(`{
		"name":     "%s",
		"email":    "%s",
		"password": "rahasia123",
		"role":     "%s"
	}`, name, email, role), "")

	resp := doPostAuth(router, "/api/v1/auth/login", fmt.Sprintf(`{
		"email":    "%s",
		"password": "rahasia123"
	}`, email), "")
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	tokens, _ := data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)
	return token
}

// createReport membuat laporan dan mengembalikan ID-nya
func createReport(token, reportType, gender string, extraFields ...string) string {
	router := getReportRouter()
	extra := ""
	if len(extraFields) > 0 {
		extra = "," + extraFields[0]
	}
	resp := doPostAuth(router, "/api/v1/reports", fmt.Sprintf(`{
		"type":               "%s",
		"gender":             "%s",
		"estimated_age":      65,
		"description":        "Pria tua, rambut putih, pakai baju batik biru, terlihat kebingungan",
		"last_seen_location": "Depan Pasar Petisah, Jalan Pegadaian",
		"city":               "Medan",
		"province":           "Sumatera Utara"
		%s
	}`, reportType, gender, extra), token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// createReportWithCoords membuat laporan dengan koordinat untuk test map
func createReportWithCoords(token, reportType string, lat, lng float64) string {
	router := getReportRouter()
	resp := doPostAuth(router, "/api/v1/reports", fmt.Sprintf(`{
		"type":               "%s",
		"gender":             "male",
		"estimated_age":      50,
		"description":        "Laporan dengan koordinat untuk test peta",
		"last_seen_location": "Jalan Sudirman No. 1",
		"city":               "Medan",
		"province":           "Sumatera Utara",
		"latitude":           %v,
		"longitude":          %v
	}`, reportType, lat, lng), token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// ═══════════════════════════════════════════════════════════════════════════════
// POST /reports — CREATE REPORT
// ═══════════════════════════════════════════════════════════════════════════════

// 1. Happy path — laporan missing berhasil dibuat
func TestCreateReportMissingSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder1@mail.com", "Finder Satu", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"estimated_age":      65,
		"description":        "Pria tua, rambut putih, pakai baju batik biru, terlihat kebingungan",
		"last_seen_location": "Depan Pasar Petisah, Jalan Pegadaian",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Laporan berhasil dibuat", body["message"])

	data := body["data"].(map[string]any)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "missing", data["type"])
	assert.Equal(t, "male", data["gender"])
	assert.Equal(t, "Medan", data["city"])
	assert.Equal(t, "active", data["status"])
	assert.NotEmpty(t, data["whatsapp_share_url"])
	assert.NotNil(t, data["reporter"])
}

// 2. Happy path — laporan found berhasil dibuat
func TestCreateReportFoundSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder2@mail.com", "Finder Dua", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "found",
		"gender":             "female",
		"description":        "Perempuan lansia, rambut pendek, pakai daster motif bunga",
		"last_seen_location": "Depan RS Adam Malik",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 201, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Equal(t, "found", data["type"])
	assert.Equal(t, "female", data["gender"])
}

// 3. Happy path — dengan nama (opsional)
func TestCreateReportWithName(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder3@mail.com", "Finder Tiga", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"name":               "Pak Rudi",
		"gender":             "male",
		"estimated_age":      70,
		"description":        "Pak Rudi menghilang sejak 3 hari lalu dari rumah",
		"last_seen_location": "Jalan Gatot Subroto No. 5",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 201, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Equal(t, "Pak Rudi", data["name"])
}

// 4. Happy path — dengan koordinat GPS
func TestCreateReportWithCoordinates(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder4@mail.com", "Finder Empat", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "found",
		"gender":             "unknown",
		"description":        "Seseorang ditemukan di pinggir jalan dalam kondisi bingung",
		"last_seen_location": "Jalan Imam Bonjol",
		"city":               "Medan",
		"province":           "Sumatera Utara",
		"latitude":           3.5952,
		"longitude":          98.6722
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 201, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.NotNil(t, data["latitude"])
	assert.NotNil(t, data["longitude"])
	assert.InDelta(t, 3.5952, data["latitude"].(float64), 0.0001)
}

// 5. Happy path — gender unknown valid
func TestCreateReportGenderUnknown(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder5@mail.com", "Finder Lima", "seeker")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "found",
		"gender":             "unknown",
		"description":        "Tidak bisa diidentifikasi jenis kelaminnya karena kondisi",
		"last_seen_location": "Terminal Amplas",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 201, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Equal(t, "unknown", data["gender"])
}

// 6. Tanpa token → 401
func TestCreateReportUnauthorized(t *testing.T) {
	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"description":        "Deskripsi laporan yang cukup panjang",
		"last_seen_location": "Lokasi terakhir",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, "")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 7. Type tidak valid → 400
func TestCreateReportInvalidType(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder6@mail.com", "Finder Enam", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "unknown_type",
		"gender":             "male",
		"description":        "Deskripsi cukup panjang untuk memenuhi validasi minimum",
		"last_seen_location": "Lokasi terakhir terlihat",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 8. Gender tidak valid → 400
func TestCreateReportInvalidGender(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder7@mail.com", "Finder Tujuh", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "other",
		"description":        "Deskripsi cukup panjang untuk memenuhi validasi minimum",
		"last_seen_location": "Lokasi terakhir terlihat",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 9. Deskripsi kurang dari 10 karakter → 400
func TestCreateReportDescriptionTooShort(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder8@mail.com", "Finder Delapan", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"description":        "Singkat",
		"last_seen_location": "Lokasi terakhir",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 10. Field city kosong → 400
func TestCreateReportMissingCity(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder9@mail.com", "Finder Sembilan", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"description":        "Deskripsi yang cukup panjang untuk test",
		"last_seen_location": "Lokasi terakhir",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 11. Field province kosong → 400
func TestCreateReportMissingProvince(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder10@mail.com", "Finder Sepuluh", "seeker")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"description":        "Deskripsi yang cukup panjang untuk test",
		"last_seen_location": "Lokasi terakhir",
		"city":               "Medan"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 12. estimated_age melebihi 120 → 400
func TestCreateReportAgeOutOfRange(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder11@mail.com", "Finder Sebelas", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"estimated_age":      999,
		"description":        "Deskripsi yang cukup panjang untuk test validasi",
		"last_seen_location": "Lokasi terakhir terlihat",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 13. Body kosong → 400
func TestCreateReportEmptyBody(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder12@mail.com", "Finder Duabelas", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 14. JSON tidak valid → 400
func TestCreateReportInvalidJSON(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder13@mail.com", "Finder Tigabelas", "finder")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{ invalid json }`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 15. Verifikasi status default = active
func TestCreateReportDefaultStatusActive(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder14@mail.com", "Finder Empatbelas", "finder")

	id := createReport(token, "missing", "male")
	assert.NotEmpty(t, id)

	var report model.Report
	testDB.Where("id = ?", id).First(&report)
	assert.Equal(t, model.ReportStatusActive, report.Status)
}

// 16. Verifikasi reporter_id tersimpan sesuai user yang login
func TestCreateReportReporterIDCorrect(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder15@mail.com", "Finder Limabelas", "finder")

	id := createReport(token, "missing", "female")

	var report model.Report
	testDB.Where("id = ?", id).First(&report)

	var user model.User
	testDB.Where("email = ?", "finder15@mail.com").First(&user)

	assert.Equal(t, user.ID, report.ReporterID)
}

// 17. Verifikasi whatsapp_share_url ada dan berformat benar
func TestCreateReportWhatsappURLPresent(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("finder16@mail.com", "Finder Enambelas", "seeker")

	resp := doPostAuth(getReportRouter(), "/api/v1/reports", `{
		"type":               "missing",
		"gender":             "male",
		"description":        "Pria dewasa hilang sejak kemarin, terakhir terlihat di pasar",
		"last_seen_location": "Pasar Sentral Medan",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	waURL, _ := data["whatsapp_share_url"].(string)
	assert.True(t, strings.HasPrefix(waURL, "https://wa.me/?text="))
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /reports — LIST ALL REPORTS
// ═══════════════════════════════════════════════════════════════════════════════

// 18. Public — list laporan tanpa token berhasil
func TestGetAllReportsPublic(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list1@mail.com", "List Satu", "finder")
	createReport(token, "missing", "male")
	createReport(token, "found", "female")

	resp := doGet(getReportRouter(), "/api/v1/reports")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	assert.GreaterOrEqual(t, len(reports), 2)

	meta := data["meta"].(map[string]any)
	assert.NotNil(t, meta["total"])
	assert.NotNil(t, meta["page"])
	assert.NotNil(t, meta["limit"])
	assert.NotNil(t, meta["total_pages"])
}

// 19. Filter by type=missing
func TestGetAllReportsFilterByType(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list2@mail.com", "List Dua", "finder")
	createReport(token, "missing", "male")
	createReport(token, "missing", "female")
	createReport(token, "found", "male")

	resp := doGet(getReportRouter(), "/api/v1/reports?type=missing")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)

	for _, r := range reports {
		report := r.(map[string]any)
		assert.Equal(t, "missing", report["type"])
	}
}

// 20. Filter by type=found
func TestGetAllReportsFilterByTypeFound(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list3@mail.com", "List Tiga", "finder")
	createReport(token, "found", "female")
	createReport(token, "missing", "male")

	resp := doGet(getReportRouter(), "/api/v1/reports?type=found")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	for _, r := range reports {
		assert.Equal(t, "found", r.(map[string]any)["type"])
	}
}

// 21. Filter by city
func TestGetAllReportsFilterByCity(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list4@mail.com", "List Empat", "seeker")
	createReport(token, "missing", "male")

	resp := doGet(getReportRouter(), "/api/v1/reports?city=Medan")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	for _, r := range reports {
		assert.Equal(t, "Medan", r.(map[string]any)["city"])
	}
}

// 22. Filter by gender
func TestGetAllReportsFilterByGender(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list5@mail.com", "List Lima", "finder")
	createReport(token, "missing", "male")
	createReport(token, "found", "female")

	resp := doGet(getReportRouter(), "/api/v1/reports?gender=male")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	for _, r := range reports {
		assert.Equal(t, "male", r.(map[string]any)["gender"])
	}
}

// 23. Filter by age range
func TestGetAllReportsFilterByAgeRange(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list6@mail.com", "List Enam", "finder")
	createReport(token, "missing", "male")

	resp := doGet(getReportRouter(), "/api/v1/reports?age_min=50&age_max=70")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	for _, r := range reports {
		age := r.(map[string]any)["estimated_age"]
		if age != nil {
			ageFloat := age.(float64)
			assert.GreaterOrEqual(t, ageFloat, float64(50))
			assert.LessOrEqual(t, ageFloat, float64(70))
		}
	}
}

// 24. Full-text search dengan ?q=
func TestGetAllReportsFullTextSearch(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list7@mail.com", "List Tujuh", "finder")
	createReport(token, "missing", "male") // deskripsi: "batik biru"

	resp := doGet(getReportRouter(), "/api/v1/reports?q=batik+biru")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	assert.GreaterOrEqual(t, len(reports), 1)
}

// 25. Pagination — page dan limit berfungsi
func TestGetAllReportsPagination(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list8@mail.com", "List Delapan", "finder")
	for i := 0; i < 5; i++ {
		createReport(token, "missing", "male")
	}

	resp := doGet(getReportRouter(), "/api/v1/reports?page=1&limit=2")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	assert.Equal(t, 2, len(reports))

	meta := data["meta"].(map[string]any)
	assert.Equal(t, float64(1), meta["page"])
	assert.Equal(t, float64(2), meta["limit"])
	assert.GreaterOrEqual(t, meta["total_pages"].(float64), float64(3))
}

// 26. Default hanya tampilkan status=active
func TestGetAllReportsDefaultStatusActive(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list9@mail.com", "List Sembilan", "finder")
	id := createReport(token, "missing", "male")

	// Ubah satu laporan jadi resolved
	testDB.Model(&model.Report{}).Where("id = ?", id).Update("status", "resolved")

	// Buat satu lagi yang aktif
	createReport(token, "found", "female")

	resp := doGet(getReportRouter(), "/api/v1/reports")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	for _, r := range reports {
		assert.Equal(t, "active", r.(map[string]any)["status"])
	}
}

// 27. Filter status=resolved
func TestGetAllReportsFilterByStatusResolved(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("list10@mail.com", "List Sepuluh", "finder")
	id := createReport(token, "missing", "male")
	testDB.Model(&model.Report{}).Where("id = ?", id).Update("status", "resolved")

	resp := doGet(getReportRouter(), "/api/v1/reports?status=resolved")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	for _, r := range reports {
		assert.Equal(t, "resolved", r.(map[string]any)["status"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /reports/:id — DETAIL REPORT
// ═══════════════════════════════════════════════════════════════════════════════

// 28. Detail laporan berhasil (public)
func TestGetReportByIDSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("detail1@mail.com", "Detail Satu", "finder")
	id := createReport(token, "missing", "male")

	resp := doGet(getReportRouter(), "/api/v1/reports/"+id)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, id, data["id"])
	assert.Equal(t, "missing", data["type"])
	assert.NotNil(t, data["reporter"])
	assert.NotEmpty(t, data["whatsapp_share_url"])
}

// 29. ID tidak valid (bukan UUID) → 400
func TestGetReportByIDInvalidID(t *testing.T) {
	resp := doGet(getReportRouter(), "/api/v1/reports/bukan-uuid")
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 30. ID tidak ditemukan → 404
func TestGetReportByIDNotFound(t *testing.T) {
	resp := doGet(getReportRouter(), "/api/v1/reports/00000000-0000-0000-0000-000000000000")
	body := parseBodyReport(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /reports/my — MY REPORTS
// ═══════════════════════════════════════════════════════════════════════════════

// 31. My reports — hanya laporan milik user yang login
func TestGetMyReportsSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token1 := registerAndLogin("my1@mail.com", "My Satu", "finder")
	token2 := registerAndLogin("my2@mail.com", "My Dua", "seeker")

	createReport(token1, "missing", "male")
	createReport(token1, "found", "female")
	createReport(token2, "missing", "male") // milik user lain

	resp := doGet(getReportRouter(), "/api/v1/reports/my", token1)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	assert.Equal(t, 2, len(reports))

	// Pastikan semua laporan milik user1
	var user1 model.User
	testDB.Where("email = ?", "my1@mail.com").First(&user1)
	for _, r := range reports {
		report := r.(map[string]any)
		assert.Equal(t, user1.ID.String(), report["reporter_id"])
	}
}

// 32. My reports tanpa token → 401
func TestGetMyReportsUnauthorized(t *testing.T) {
	resp := doGet(getReportRouter(), "/api/v1/reports/my")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 33. My reports ketika tidak punya laporan → 200, array kosong
func TestGetMyReportsEmpty(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("my3@mail.com", "My Tiga", "volunteer")

	resp := doGet(getReportRouter(), "/api/v1/reports/my", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	assert.Equal(t, 0, len(reports))
}

// 34. My reports dengan pagination
func TestGetMyReportsPagination(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("my4@mail.com", "My Empat", "finder")
	for i := 0; i < 4; i++ {
		createReport(token, "missing", "male")
	}

	resp := doGet(getReportRouter(), "/api/v1/reports/my?page=1&limit=2", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	reports := data["reports"].([]any)
	assert.Equal(t, 2, len(reports))

	meta := data["meta"].(map[string]any)
	assert.Equal(t, float64(4), meta["total"])
	assert.Equal(t, float64(2), meta["total_pages"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// PUT /reports/:id — UPDATE REPORT
// ═══════════════════════════════════════════════════════════════════════════════

// 35. Update berhasil oleh pelapor
func TestUpdateReportSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update1@mail.com", "Update Satu", "finder")
	id := createReport(token, "missing", "male")

	resp := doPut(getReportRouter(), "/api/v1/reports/"+id, `{
		"city":        "Binjai",
		"province":    "Sumatera Utara",
		"description": "Deskripsi sudah diperbarui dengan informasi terbaru yang lebih lengkap"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Laporan berhasil diperbarui", body["message"])

	data := body["data"].(map[string]any)
	assert.Equal(t, "Binjai", data["city"])
}

// 36. Update status jadi resolved
func TestUpdateReportStatusResolved(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update2@mail.com", "Update Dua", "seeker")
	id := createReport(token, "missing", "male")

	resp := doPut(getReportRouter(), "/api/v1/reports/"+id, `{
		"status": "resolved"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Equal(t, "resolved", data["status"])
}

// 37. Update oleh user lain (bukan pelapor) → 403
func TestUpdateReportForbidden(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token1 := registerAndLogin("update3@mail.com", "Update Tiga", "finder")
	token2 := registerAndLogin("update4@mail.com", "Update Empat", "seeker")
	id := createReport(token1, "missing", "male")

	resp := doPut(getReportRouter(), "/api/v1/reports/"+id, `{
		"city": "Binjai"
	}`, token2)
	body := parseBodyReport(resp)

	assert.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "FORBIDDEN", body["status"])
}

// 38. Update tanpa token → 401
func TestUpdateReportUnauthorized(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update5@mail.com", "Update Lima", "finder")
	id := createReport(token, "missing", "male")

	resp := doPut(getReportRouter(), "/api/v1/reports/"+id, `{
		"city": "Binjai"
	}`, "")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 39. Update laporan yang tidak ada → 404
func TestUpdateReportNotFound(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update6@mail.com", "Update Enam", "finder")

	resp := doPut(getReportRouter(), "/api/v1/reports/00000000-0000-0000-0000-000000000000", `{
		"city": "Binjai"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 40. Update ID tidak valid (bukan UUID) → 400
func TestUpdateReportInvalidID(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update7@mail.com", "Update Tujuh", "finder")

	resp := doPut(getReportRouter(), "/api/v1/reports/bukan-uuid", `{
		"city": "Binjai"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 41. Update gender tidak valid → 400
func TestUpdateReportInvalidGender(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update8@mail.com", "Update Delapan", "finder")
	id := createReport(token, "missing", "male")

	resp := doPut(getReportRouter(), "/api/v1/reports/"+id, `{
		"gender": "invalid_gender"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 42. Update hanya field yang dikirim (partial update / PATCH behaviour)
func TestUpdateReportPartialUpdate(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("update9@mail.com", "Update Sembilan", "finder")
	id := createReport(token, "missing", "male")

	// Hanya update city, field lain tidak berubah
	resp := doPut(getReportRouter(), "/api/v1/reports/"+id, `{
		"city": "Deli Serdang"
	}`, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	assert.Equal(t, "Deli Serdang", data["city"])
	// Field lain tetap
	assert.Equal(t, "missing", data["type"])
	assert.Equal(t, "male", data["gender"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// DELETE /reports/:id — DELETE REPORT
// ═══════════════════════════════════════════════════════════════════════════════

// 43. Delete berhasil oleh pelapor
func TestDeleteReportSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("delete1@mail.com", "Delete Satu", "finder")
	id := createReport(token, "missing", "male")

	resp := doDelete(getReportRouter(), "/api/v1/reports/"+id, token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.Equal(t, "Laporan berhasil dihapus", body["message"])
}

// 44. Verifikasi laporan benar-benar terhapus dari DB
func TestDeleteReportVerifyInDB(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("delete2@mail.com", "Delete Dua", "finder")
	id := createReport(token, "missing", "male")

	doDelete(getReportRouter(), "/api/v1/reports/"+id, token)

	// GET setelah delete harus 404
	resp := doGet(getReportRouter(), "/api/v1/reports/"+id)
	assert.Equal(t, 404, resp.StatusCode)
}

// 45. Delete oleh user lain → 403
func TestDeleteReportForbidden(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token1 := registerAndLogin("delete3@mail.com", "Delete Tiga", "finder")
	token2 := registerAndLogin("delete4@mail.com", "Delete Empat", "seeker")
	id := createReport(token1, "missing", "male")

	resp := doDelete(getReportRouter(), "/api/v1/reports/"+id, token2)
	body := parseBodyReport(resp)

	assert.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "FORBIDDEN", body["status"])
}

// 46. Delete tanpa token → 401
func TestDeleteReportUnauthorized(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("delete5@mail.com", "Delete Lima", "finder")
	id := createReport(token, "missing", "male")

	resp := doDelete(getReportRouter(), "/api/v1/reports/"+id, "")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 47. Delete laporan yang tidak ada → 404
func TestDeleteReportNotFound(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("delete6@mail.com", "Delete Enam", "finder")

	resp := doDelete(getReportRouter(), "/api/v1/reports/00000000-0000-0000-0000-000000000000", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 48. Delete ID tidak valid → 400
func TestDeleteReportInvalidID(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("delete7@mail.com", "Delete Tujuh", "finder")

	resp := doDelete(getReportRouter(), "/api/v1/reports/bukan-uuid", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// POST /reports/:id/photo — UPLOAD PHOTO
// ═══════════════════════════════════════════════════════════════════════════════

// Helper untuk multipart upload
func doUploadPhoto(router *gin.Engine, url, token string, fileContent []byte, filename string) *http.Response {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("photo", filename)
	part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

// 49. Upload foto tanpa token → 401
func TestUploadPhotoUnauthorized(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("photo1@mail.com", "Photo Satu", "finder")
	id := createReport(token, "missing", "male")

	fakeImg := make([]byte, 100)
	resp := doUploadPhoto(getReportRouter(), "/api/v1/reports/"+id+"/photo", "", fakeImg, "foto.jpg")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 50. Upload foto oleh user lain (bukan pelapor) → 403
func TestUploadPhotoForbidden(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token1 := registerAndLogin("photo2@mail.com", "Photo Dua", "finder")
	token2 := registerAndLogin("photo3@mail.com", "Photo Tiga", "seeker")
	id := createReport(token1, "missing", "male")

	fakeImg := make([]byte, 100)
	resp := doUploadPhoto(getReportRouter(), "/api/v1/reports/"+id+"/photo", token2, fakeImg, "foto.jpg")
	body := parseBodyReport(resp)

	assert.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "FORBIDDEN", body["status"])
}

// 51. Upload foto laporan tidak ada → 404
func TestUploadPhotoReportNotFound(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("photo4@mail.com", "Photo Empat", "finder")

	fakeImg := make([]byte, 100)
	resp := doUploadPhoto(getReportRouter(),
		"/api/v1/reports/00000000-0000-0000-0000-000000000000/photo",
		token, fakeImg, "foto.jpg")
	body := parseBodyReport(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 52. Upload tanpa file → 400
func TestUploadPhotoMissingFile(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("photo5@mail.com", "Photo Lima", "finder")
	id := createReport(token, "missing", "male")

	// POST tanpa multipart file
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reports/"+id+"/photo", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	getReportRouter().ServeHTTP(rec, req)
	resp := rec.Result()
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 53. Upload file terlalu besar → 400
func TestUploadPhotoFileTooLarge(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("photo6@mail.com", "Photo Enam", "finder")
	id := createReport(token, "missing", "male")

	// Buat file >5MB
	bigFile := make([]byte, 6*1024*1024)
	resp := doUploadPhoto(getReportRouter(), "/api/v1/reports/"+id+"/photo", token, bigFile, "besar.jpg")
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 54. Upload format file tidak didukung → 400
func TestUploadPhotoInvalidFormat(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("photo7@mail.com", "Photo Tujuh", "finder")
	id := createReport(token, "missing", "male")

	fakeFile := []byte("ini bukan gambar")
	resp := doUploadPhoto(getReportRouter(), "/api/v1/reports/"+id+"/photo", token, fakeFile, "dokumen.pdf")
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 55. Upload ID tidak valid → 400
func TestUploadPhotoInvalidID(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("photo8@mail.com", "Photo Delapan", "finder")

	fakeImg := make([]byte, 100)
	resp := doUploadPhoto(getReportRouter(), "/api/v1/reports/bukan-uuid/photo", token, fakeImg, "foto.jpg")
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /map/pins — MAP PINS
// ═══════════════════════════════════════════════════════════════════════════════

// 56. Map pins berhasil (public)
func TestGetMapPinsSuccess(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("map1@mail.com", "Map Satu", "finder")
	createReportWithCoords(token, "missing", 3.5952, 98.6722)
	createReportWithCoords(token, "found", 3.6100, 98.7000)

	resp := doGet(getReportRouter(), "/api/v1/map/pins")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	pins := data["pins"].([]any)
	assert.GreaterOrEqual(t, len(pins), 2)
	assert.NotNil(t, data["total"])

	// Verifikasi struktur pin (hanya field yang diperlukan)
	pin := pins[0].(map[string]any)
	assert.NotEmpty(t, pin["id"])
	assert.NotEmpty(t, pin["type"])
	assert.NotEmpty(t, pin["gender"])
	assert.NotNil(t, pin["latitude"])
	assert.NotNil(t, pin["longitude"])
	assert.NotEmpty(t, pin["city"])
}

// 57. Laporan tanpa koordinat tidak muncul di pins
func TestGetMapPinsExcludesNoCoords(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("map2@mail.com", "Map Dua", "seeker")

	// Laporan DENGAN koordinat
	createReportWithCoords(token, "missing", 3.5952, 98.6722)

	// Laporan TANPA koordinat (tidak boleh muncul di pins)
	createReport(token, "found", "female")

	resp := doGet(getReportRouter(), "/api/v1/map/pins")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	pins := data["pins"].([]any)

	for _, p := range pins {
		pin := p.(map[string]any)
		assert.NotNil(t, pin["latitude"], "semua pin harus punya koordinat")
		assert.NotNil(t, pin["longitude"], "semua pin harus punya koordinat")
	}
}

// 58. Filter pins by type=missing
func TestGetMapPinsFilterByType(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("map3@mail.com", "Map Tiga", "finder")
	createReportWithCoords(token, "missing", 3.5952, 98.6722)
	createReportWithCoords(token, "found", 3.6100, 98.7000)

	resp := doGet(getReportRouter(), "/api/v1/map/pins?type=missing")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	pins := data["pins"].([]any)
	for _, p := range pins {
		assert.Equal(t, "missing", p.(map[string]any)["type"])
	}
}

// 59. Filter pins by bounds
func TestGetMapPinsBoundsFilter(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("map4@mail.com", "Map Empat", "finder")

	// Pin dalam bounds Medan
	createReportWithCoords(token, "missing", 3.5952, 98.6722)
	// Pin di luar bounds (Jakarta)
	createReportWithCoords(token, "found", -6.2088, 106.8456)

	// Bounds Medan area
	resp := doGet(getReportRouter(), "/api/v1/map/pins?bounds=3.4,98.5,3.7,98.8")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	pins := data["pins"].([]any)

	// Hanya pin Medan yang masuk
	for _, p := range pins {
		pin := p.(map[string]any)
		lat := pin["latitude"].(float64)
		lng := pin["longitude"].(float64)
		assert.GreaterOrEqual(t, lat, 3.4)
		assert.LessOrEqual(t, lat, 3.7)
		assert.GreaterOrEqual(t, lng, 98.5)
		assert.LessOrEqual(t, lng, 98.8)
	}
}

// 60. Pins hanya menampilkan laporan aktif (bukan resolved)
func TestGetMapPinsOnlyActive(t *testing.T) {
	truncateReports(testDB)
	truncateTables(testDB)
	token := registerAndLogin("map5@mail.com", "Map Lima", "finder")

	idActive := createReportWithCoords(token, "missing", 3.5952, 98.6722)
	idResolved := createReportWithCoords(token, "found", 3.6000, 98.6800)

	// Set satu laporan jadi resolved
	testDB.Model(&model.Report{}).Where("id = ?", idResolved).Update("status", "resolved")

	resp := doGet(getReportRouter(), "/api/v1/map/pins")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	pins := data["pins"].([]any)

	// Cek laporan resolved tidak ada di pins
	for _, p := range pins {
		pin := p.(map[string]any)
		assert.NotEqual(t, idResolved, pin["id"], "laporan resolved tidak boleh muncul di map")
	}

	// Laporan active harus ada
	found := false
	for _, p := range pins {
		if p.(map[string]any)["id"] == idActive {
			found = true
		}
	}
	assert.True(t, found, "laporan active harus muncul di map")

	// Hindari unused variable warning
	_ = os.Getenv("DB_HOST")
}
