package test

import (
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

func setupStatsRouter(db *gorm.DB) *gin.Engine {
	// User layer (dibutuhkan untuk seed data)
	validate := validator.New()
	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, validate)
	userHandler := handler.NewUserHandlerImpl(userUsecase)

	// Report layer
	reportRepo := repository.NewReportRepository(db)
	reportUsecase := usecase.NewReportUsecase(reportRepo, validate, nil)
	reportHandler := handler.NewReportHandlerImpl(reportUsecase)

	// Stats layer
	statsRepo := repository.NewStatsRepository(db)
	statsUsecase := usecase.NewStatsUsecase(statsRepo)
	statsHandler := handler.NewStatsHandlerImpl(statsUsecase)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.ErrorRecovery())

	api := r.Group("/api/v1")

	// Auth (untuk seed token)
	auth := api.Group("/auth")
	auth.POST("/register", userHandler.Create)
	auth.POST("/login", userHandler.Login)

	// Reports (untuk seed data laporan)
	reports := api.Group("/reports")
	reports.GET("", reportHandler.GetAll)
	reportsPrivate := reports.Group("")
	reportsPrivate.Use(middleware.AuthMiddleware())
	reportsPrivate.POST("", reportHandler.Create)
	reportsPrivate.PUT("/:id", reportHandler.Update)

	// Stats — endpoint yang ditest
	api.GET("/stats", statsHandler.GetStats)

	return r
}

func truncateStats(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE reports RESTART IDENTITY CASCADE")
	db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
}

var statsRouter *gin.Engine

func getStatsRouter() *gin.Engine {
	if statsRouter == nil {
		statsRouter = setupStatsRouter(testDB)
	}
	return statsRouter
}

// registerAndLoginStats mendaftarkan user dan mengembalikan token
func registerAndLoginStats(email, name, role string) string {
	router := getStatsRouter()
	doPostAuth(router, "/api/v1/auth/register", `{
		"name":     "`+name+`",
		"email":    "`+email+`",
		"password": "rahasia123",
		"role":     "`+role+`"
	}`, "")

	resp := doPostAuth(router, "/api/v1/auth/login", `{
		"email":    "`+email+`",
		"password": "rahasia123"
	}`, "")
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	tokens, _ := data["tokens"].(map[string]any)
	token, _ := tokens["access_token"].(string)
	return token
}

// createReportStats membuat laporan melalui statsRouter
func createReportStats(token, reportType, gender string) string {
	router := getStatsRouter()
	resp := doPostAuth(router, "/api/v1/reports", `{
		"type":               "`+reportType+`",
		"gender":             "`+gender+`",
		"estimated_age":      40,
		"description":        "Deskripsi laporan untuk test stats endpoint",
		"last_seen_location": "Jalan Sudirman No. 1",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// createReportStatsCity membuat laporan dengan kota tertentu
func createReportStatsCity(token, reportType, city string) string {
	router := getStatsRouter()
	resp := doPostAuth(router, "/api/v1/reports", `{
		"type":               "`+reportType+`",
		"gender":             "female",
		"estimated_age":      30,
		"description":        "Deskripsi laporan untuk test unique cities",
		"last_seen_location": "Jalan Utama No. 5",
		"city":               "`+city+`",
		"province":           "Sumatera Utara"
	}`, token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — HAPPY PATH
// ═══════════════════════════════════════════════════════════════════════════════

// 1. Happy path — endpoint publik berhasil tanpa auth
func TestGetStatsSuccess(t *testing.T) {
	truncateStats(testDB)

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.NotNil(t, body["data"])

	data := body["data"].(map[string]any)
	assert.Contains(t, data, "active_reports")
	assert.Contains(t, data, "total_volunteers")
	assert.Contains(t, data, "total_resolved")
	assert.Contains(t, data, "resolved_last_24h")
	assert.Contains(t, data, "unique_cities")
}

// 2. Semua nilai bertipe number (bukan null/string)
func TestGetStatsFieldTypes(t *testing.T) {
	truncateStats(testDB)

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)

	// JSON number di-decode sebagai float64 oleh Go
	_, okActive := data["active_reports"].(float64)
	_, okVol := data["total_volunteers"].(float64)
	_, okTotalResolved := data["total_resolved"].(float64)
	_, okResolved := data["resolved_last_24h"].(float64)
	_, okCities := data["unique_cities"].(float64)

	assert.True(t, okActive, "active_reports harus bertipe number")
	assert.True(t, okVol, "total_volunteers harus bertipe number")
	assert.True(t, okTotalResolved, "total_resolved harus bertipe number")
	assert.True(t, okResolved, "resolved_last_24h harus bertipe number")
	assert.True(t, okCities, "unique_cities harus bertipe number")
}

// 3. DB kosong — semua nilai 0
func TestGetStatsAllZeroWhenEmpty(t *testing.T) {
	truncateStats(testDB)

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["active_reports"])
	assert.Equal(t, float64(0), data["total_volunteers"])
	assert.Equal(t, float64(0), data["total_resolved"])
	assert.Equal(t, float64(0), data["resolved_last_24h"])
	assert.Equal(t, float64(0), data["unique_cities"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — active_reports
// ═══════════════════════════════════════════════════════════════════════════════

// 4. active_reports menghitung laporan dengan status active
func TestGetStatsActiveReportsCount(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("stats1@mail.com", "Stats Satu", "finder")

	createReportStats(token, "missing", "male")
	createReportStats(token, "found", "female")
	createReportStats(token, "missing", "unknown")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(3), data["active_reports"])
}

// 5. active_reports tidak menghitung laporan resolved
func TestGetStatsActiveReportsExcludesResolved(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("stats2@mail.com", "Stats Dua", "seeker")

	createReportStats(token, "missing", "male")
	idResolved := createReportStats(token, "found", "female")

	// Set satu laporan jadi resolved
	testDB.Model(&model.Report{}).Where("id = ?", idResolved).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(1), data["active_reports"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — total_volunteers
// ═══════════════════════════════════════════════════════════════════════════════

// 6. total_volunteers menghitung user dengan role volunteer
func TestGetStatsTotalVolunteersCount(t *testing.T) {
	truncateStats(testDB)

	registerAndLoginStats("vol1@mail.com", "Relawan Satu", "volunteer")
	registerAndLoginStats("vol2@mail.com", "Relawan Dua", "volunteer")
	registerAndLoginStats("vol3@mail.com", "Relawan Tiga", "volunteer")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(3), data["total_volunteers"])
}

// 7. total_volunteers tidak menghitung finder/seeker
func TestGetStatsTotalVolunteersExcludesOtherRoles(t *testing.T) {
	truncateStats(testDB)

	registerAndLoginStats("finder1@mail.com", "Finder Satu", "finder")
	registerAndLoginStats("seeker1@mail.com", "Seeker Satu", "seeker")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total_volunteers"])
}

// 8. total_volunteers bertambah saat volunteer baru daftar
func TestGetStatsTotalVolunteersIncrement(t *testing.T) {
	truncateStats(testDB)

	resp1 := doGet(getStatsRouter(), "/api/v1/stats")
	body1 := parseBodyReport(resp1)
	before := body1["data"].(map[string]any)["total_volunteers"].(float64)

	registerAndLoginStats("vol_new@mail.com", "Relawan Baru", "volunteer")

	resp2 := doGet(getStatsRouter(), "/api/v1/stats")
	body2 := parseBodyReport(resp2)
	after := body2["data"].(map[string]any)["total_volunteers"].(float64)

	assert.Equal(t, before+1, after)
}

// 9. total_volunteers tidak berubah saat finder/seeker daftar
func TestGetStatsTotalVolunteersNotAffectedByOtherRoles(t *testing.T) {
	truncateStats(testDB)
	registerAndLoginStats("vol_base@mail.com", "Relawan Base", "volunteer")

	resp1 := doGet(getStatsRouter(), "/api/v1/stats")
	body1 := parseBodyReport(resp1)
	before := body1["data"].(map[string]any)["total_volunteers"].(float64)

	registerAndLoginStats("finder_new@mail.com", "Finder Baru", "finder")
	registerAndLoginStats("seeker_new@mail.com", "Seeker Baru", "seeker")

	resp2 := doGet(getStatsRouter(), "/api/v1/stats")
	body2 := parseBodyReport(resp2)
	after := body2["data"].(map[string]any)["total_volunteers"].(float64)

	assert.Equal(t, before, after, "total_volunteers tidak boleh berubah saat non-volunteer daftar")
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — total_resolved
// ═══════════════════════════════════════════════════════════════════════════════

// 10. total_resolved menghitung semua laporan dengan status resolved
func TestGetStatsTotalResolvedCount(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("tr1@mail.com", "Total Resolved Satu", "finder")

	id1 := createReportStats(token, "missing", "male")
	id2 := createReportStats(token, "found", "female")
	createReportStats(token, "missing", "unknown") // tetap active

	testDB.Model(&model.Report{}).Where("id = ?", id1).Update("status", "resolved")
	testDB.Model(&model.Report{}).Where("id = ?", id2).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(2), data["total_resolved"])
}

// 11. total_resolved = 0 saat tidak ada laporan resolved
func TestGetStatsTotalResolvedZeroWhenNone(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("tr2@mail.com", "Total Resolved Dua", "finder")

	createReportStats(token, "missing", "male")
	createReportStats(token, "found", "female")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total_resolved"])
}

// 12. total_resolved menghitung laporan resolved kapan pun (tidak terbatas 24h)
func TestGetStatsTotalResolvedIncludesOldResolved(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("tr3@mail.com", "Total Resolved Tiga", "finder")

	id := createReportStats(token, "missing", "male")

	// Paksa updated_at ke 2 hari lalu
	testDB.Model(&model.Report{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":     "resolved",
			"updated_at": testDB.NowFunc().AddDate(0, 0, -2),
		})

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	// total_resolved harus menghitung ini meski lebih dari 24h
	assert.Equal(t, float64(1), data["total_resolved"])
	// resolved_last_24h tidak menghitung ini
	assert.Equal(t, float64(0), data["resolved_last_24h"])
}

// 13. total_resolved tidak menghitung laporan active
func TestGetStatsTotalResolvedExcludesActive(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("tr4@mail.com", "Total Resolved Empat", "seeker")

	createReportStats(token, "missing", "male")
	createReportStats(token, "found", "female")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total_resolved"])
	assert.Equal(t, float64(2), data["active_reports"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — resolved_last_24h
// ═══════════════════════════════════════════════════════════════════════════════

// 14. resolved_last_24h menghitung laporan yang baru saja resolved
func TestGetStatsResolvedLast24h(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("res1@mail.com", "Resolved Satu", "finder")

	id1 := createReportStats(token, "missing", "male")
	id2 := createReportStats(token, "found", "female")
	createReportStats(token, "missing", "unknown") // tetap active

	testDB.Model(&model.Report{}).Where("id = ?", id1).Update("status", "resolved")
	testDB.Model(&model.Report{}).Where("id = ?", id2).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(2), data["resolved_last_24h"])
}

// 15. resolved_last_24h tidak menghitung laporan yang masih active
func TestGetStatsResolvedLast24hExcludesActive(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("res2@mail.com", "Resolved Dua", "seeker")

	createReportStats(token, "missing", "male")
	createReportStats(token, "found", "female")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["resolved_last_24h"])
}

// 16. resolved_last_24h tidak menghitung laporan resolved lebih dari 24 jam lalu
func TestGetStatsResolvedLast24hExcludesOldResolved(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("res3@mail.com", "Resolved Tiga", "finder")

	id := createReportStats(token, "missing", "male")

	testDB.Model(&model.Report{}).Where("id = ?", id).
		Updates(map[string]any{
			"status":     "resolved",
			"updated_at": testDB.NowFunc().AddDate(0, 0, -2),
		})

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["resolved_last_24h"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — unique_cities
// ═══════════════════════════════════════════════════════════════════════════════

// 17. unique_cities menghitung kota unik dari laporan aktif
func TestGetStatsUniqueCities(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("city1@mail.com", "City Satu", "finder")

	createReportStatsCity(token, "missing", "Medan")
	createReportStatsCity(token, "found", "Medan")         // kota sama, tidak dihitung 2x
	createReportStatsCity(token, "missing", "Binjai")
	createReportStatsCity(token, "found", "Pematangsiantar")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	// Medan, Binjai, Pematangsiantar = 3 kota unik
	assert.Equal(t, float64(3), data["unique_cities"])
}

// 18. unique_cities tidak menghitung kota dari laporan resolved
func TestGetStatsUniqueCitiesExcludesResolved(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("city2@mail.com", "City Dua", "seeker")

	createReportStatsCity(token, "missing", "Medan")
	idResolved := createReportStatsCity(token, "found", "Jakarta")

	testDB.Model(&model.Report{}).Where("id = ?", idResolved).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(1), data["unique_cities"])
}

// 19. unique_cities = 0 saat tidak ada laporan aktif
func TestGetStatsUniqueCitiesZeroWhenNoActiveReports(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("city3@mail.com", "City Tiga", "finder")

	id := createReportStatsCity(token, "missing", "Medan")
	testDB.Model(&model.Report{}).Where("id = ?", id).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(0), data["unique_cities"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /stats — AKSES & KEAMANAN
// ═══════════════════════════════════════════════════════════════════════════════

// 20. Endpoint publik — tidak butuh token
func TestGetStatsPublicNoTokenRequired(t *testing.T) {
	truncateStats(testDB)

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	assert.Equal(t, 200, resp.StatusCode, "endpoint stats harus bisa diakses tanpa auth")
}

// 21. Dengan token valid pun tetap bisa diakses (tidak di-reject)
func TestGetStatsAccessibleWithToken(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("stats_auth@mail.com", "Stats Auth", "finder")

	resp := doGet(getStatsRouter(), "/api/v1/stats", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
}

// 22. Response konsisten — harus selalu ada kelima field meski nilainya 0
func TestGetStatsResponseAlwaysHasAllFields(t *testing.T) {
	truncateStats(testDB)

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)

	fields := []string{"active_reports", "total_volunteers", "total_resolved", "resolved_last_24h", "unique_cities"}
	for _, f := range fields {
		_, exists := data[f]
		assert.True(t, exists, "field '%s' harus selalu ada di response", f)
	}
}

// 23. Nilai stats akurat setelah kombinasi laporan + user + status update
func TestGetStatsCombinedAccuracy(t *testing.T) {
	truncateStats(testDB)

	// 2 volunteer
	registerAndLoginStats("comb_vol1@mail.com", "Vol Combo Satu", "volunteer")
	registerAndLoginStats("comb_vol2@mail.com", "Vol Combo Dua", "volunteer")

	// 1 finder untuk buat laporan
	token := registerAndLoginStats("comb_finder@mail.com", "Finder Combo", "finder")

	// 3 laporan: 2 active di Medan, 1 akan di-resolved dari Binjai
	createReportStatsCity(token, "missing", "Medan")
	createReportStatsCity(token, "found", "Medan")
	idResolved := createReportStatsCity(token, "missing", "Binjai")

	testDB.Model(&model.Report{}).Where("id = ?", idResolved).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	assert.Equal(t, float64(2), data["active_reports"],    "2 laporan aktif")
	assert.Equal(t, float64(2), data["total_volunteers"],  "2 volunteer")
	assert.Equal(t, float64(1), data["total_resolved"],    "1 laporan resolved (all-time)")
	assert.Equal(t, float64(1), data["resolved_last_24h"], "1 laporan baru resolved")
	assert.Equal(t, float64(1), data["unique_cities"],     "hanya Medan yang aktif")
}

// 24. total_resolved >= resolved_last_24h selalu
func TestGetStatsTotalResolvedAlwaysGteResolvedLast24h(t *testing.T) {
	truncateStats(testDB)
	token := registerAndLoginStats("tr5@mail.com", "Total Resolved Lima", "finder")

	// 1 laporan resolved 2 hari lalu (masuk total_resolved, tidak masuk resolved_last_24h)
	idOld := createReportStats(token, "missing", "male")
	testDB.Model(&model.Report{}).Where("id = ?", idOld).
		Updates(map[string]any{
			"status":     "resolved",
			"updated_at": testDB.NowFunc().AddDate(0, 0, -2),
		})

	// 1 laporan resolved sekarang (masuk keduanya)
	idNew := createReportStats(token, "found", "female")
	testDB.Model(&model.Report{}).Where("id = ?", idNew).Update("status", "resolved")

	resp := doGet(getStatsRouter(), "/api/v1/stats")
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	totalResolved := data["total_resolved"].(float64)
	resolvedLast24h := data["resolved_last_24h"].(float64)

	assert.Equal(t, float64(2), totalResolved)
	assert.Equal(t, float64(1), resolvedLast24h)
	assert.GreaterOrEqual(t, totalResolved, resolvedLast24h,
		"total_resolved harus >= resolved_last_24h")
}