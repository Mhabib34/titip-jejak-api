package test

import (
	"context"
	"fmt"
	"net/http"
	"titip-jejak-api/internal/worker"
	"testing"

	"titip-jejak-api/internal/handler"
	"titip-jejak-api/internal/middleware"
	"titip-jejak-api/internal/repository"
	"titip-jejak-api/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ── Setup ──────────────────────────────────────────────────────────────────

func setupMatchRouter(db *gorm.DB) *gin.Engine {
	validate := validator.New()

	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, validate)
	userHandler := handler.NewUserHandlerImpl(userUsecase)

	reportRepo := repository.NewReportRepository(db)
	matchRepo := repository.NewMatchRepository(db)
	notifRepo := repository.NewNotificationRepository(db)

	// Buat worker
	matchWorker := worker.NewMatchWorker(
		reportRepo, matchRepo, notifRepo, userRepo,
		nil, // emailService boleh nil di test
		2,
	)

	// Start worker di background
	ctx := context.Background()
	go matchWorker.Start(ctx)

	// Inject worker ke usecase
	reportUsecase := usecase.NewReportUsecase(reportRepo, validate, nil, matchWorker)
	reportHandler := handler.NewReportHandlerImpl(reportUsecase)

	matchUsecase := usecase.NewMatchUsecase(matchRepo)
	matchHandler := handler.NewMatchHandlerImpl(matchUsecase)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.ErrorRecovery())

	api := r.Group("/api/v1")

	// Auth (untuk register & login)
	auth := api.Group("/auth")
	auth.POST("/register", userHandler.Create)
	auth.POST("/login", userHandler.Login)

	// Reports (untuk seed data)
	reports := api.Group("/reports")
	reportsPrivate := reports.Group("")
	reportsPrivate.Use(middleware.AuthMiddleware())
	reportsPrivate.POST("", reportHandler.Create)
	reportsPrivate.PUT("/:id", reportHandler.Update)

	// Matches
	matches := api.Group("/matches")
	matches.Use(middleware.AuthMiddleware())
	matches.GET("", matchHandler.GetAll)
	matches.GET("/:id", matchHandler.GetByID)

	return r
}

func truncateMatches(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE matches RESTART IDENTITY CASCADE")
}

var matchRouter *gin.Engine

func getMatchRouter() *gin.Engine {
	if matchRouter == nil {
		matchRouter = setupMatchRouter(testDB)
	}
	return matchRouter
}

// registerAndLoginMatch mendaftarkan user dan mengembalikan access token
func registerAndLoginMatch(email, name, role string) string {
	router := getMatchRouter()
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

// seedMatchDirect menyisipkan match langsung ke DB untuk keperluan test
func seedMatchDirect(foundReportID, missingReportID string, score int) string {
	var id string
	testDB.Raw(`
		INSERT INTO matches (found_report_id, missing_report_id, score, notified)
		VALUES (?, ?, ?, false)
		RETURNING id
	`, foundReportID, missingReportID, score).Scan(&id)
	return id
}

// createReportForMatch membuat laporan dan mengembalikan ID-nya
func createReportForMatch(token, reportType, gender string) string {
	router := getMatchRouter()
	resp := doPostAuth(router, "/api/v1/reports", fmt.Sprintf(`{
		"type":               "%s",
		"gender":             "%s",
		"estimated_age":      50,
		"description":        "Laporan untuk keperluan test match sistem",
		"last_seen_location": "Jalan Gatot Subroto",
		"city":               "Medan",
		"province":           "Sumatera Utara"
	}`, reportType, gender), token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

func createReportForMatchCity(token, reportType, gender, city string) string {
	router := getMatchRouter()
	resp := doPostAuth(router, "/api/v1/reports", fmt.Sprintf(`{
		"type":               "%s",
		"gender":             "%s",
		"estimated_age":      50,
		"description":        "Laporan untuk keperluan test match sistem",
		"last_seen_location": "Jalan Utama No. 1",
		"city":               "%s",
		"province":           "Jawa Timur"
	}`, reportType, gender, city), token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /matches — GET ALL MATCHES
// ═══════════════════════════════════════════════════════════════════════════════

// 1. Happy path — berhasil mengambil daftar match milik user
func TestGetMatchesSuccess(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("matchuser1@mail.com", "Match User Satu", "finder")
	tokenB := registerAndLoginMatch("matchuser2@mail.com", "Match User Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID := createReportForMatch(tokenB, "missing", "male")
	seedMatchDirect(foundID, missingID, 85)

	resp := doGet(getMatchRouter(), "/api/v1/matches", tokenA)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)
	assert.GreaterOrEqual(t, len(matches), 1)

	meta := data["meta"].(map[string]any)
	assert.NotNil(t, meta["page"])
	assert.NotNil(t, meta["limit"])
	assert.NotNil(t, meta["total"])
	assert.NotNil(t, meta["total_pages"])
}

// 2. Verifikasi struktur response match (field-field yang wajib ada)
func TestGetMatchesResponseStructure(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("matchstruct1@mail.com", "Struct Satu", "finder")
	tokenB := registerAndLoginMatch("matchstruct2@mail.com", "Struct Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "female")
	missingID := createReportForMatch(tokenB, "missing", "female")
	seedMatchDirect(foundID, missingID, 90)

	resp := doGet(getMatchRouter(), "/api/v1/matches", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)
	assert.NotEmpty(t, matches)

	match := matches[0].(map[string]any)
	assert.NotEmpty(t, match["id"])
	assert.NotNil(t, match["score"])
	assert.NotNil(t, match["found_report"])
	assert.NotNil(t, match["missing_report"])
	assert.NotNil(t, match["notified"])
	assert.NotEmpty(t, match["created_at"])
}

// 3. User hanya melihat match yang berkaitan dengan laporannya sendiri
// 3. User hanya melihat match yang berkaitan dengan laporannya sendiri
func TestGetMatchesOnlyOwnMatches(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("ownmatch1@mail.com", "Own Match Satu", "finder")
	tokenB := registerAndLoginMatch("ownmatch2@mail.com", "Own Match Dua", "seeker")
	tokenC := registerAndLoginMatch("ownmatch3@mail.com", "Own Match Tiga", "finder")

	// Match antara A (found) dan B (missing) — kota Medan, gender male
	foundIDA := createReportForMatch(tokenA, "found", "male")
	missingIDB := createReportForMatch(tokenB, "missing", "male")
	matchAB := seedMatchDirect(foundIDA, missingIDB, 80)
	_ = matchAB

	// Laporan C — kota BERBEDA (Surabaya) supaya worker tidak bisa
	// membuat match silang dengan laporan A atau B di Medan
	foundIDC := createReportForMatchCity(tokenC, "found", "female", "Surabaya")
	missingIDC := createReportForMatchCity(tokenC, "missing", "female", "Surabaya")
	seedMatchDirect(foundIDC, missingIDC, 75)

	// User C hanya boleh lihat match yang melibatkan laporannya sendiri
	resp := doGet(getMatchRouter(), "/api/v1/matches", tokenC)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)

	for _, m := range matches {
		match := m.(map[string]any)
		matchID := match["id"].(string)
		frID := getFoundReportID(match)
		// Tidak boleh ada match yang found_report-nya milik user A
		assert.NotEqual(t, foundIDA, frID,
			"user C tidak boleh lihat match user A: matchID=%s", matchID)
	}
}

func getFoundReportID(match map[string]any) string {
	if fr, ok := match["found_report"].(map[string]any); ok {
		id, _ := fr["id"].(string)
		return id
	}
	return ""
}

// 4. Filter min_score — hanya match dengan score >= min_score yang muncul
func TestGetMatchesFilterMinScore(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("minscore1@mail.com", "MinScore Satu", "finder")
	tokenB := registerAndLoginMatch("minscore2@mail.com", "MinScore Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID1 := createReportForMatch(tokenB, "missing", "male")
	missingID2 := createReportForMatch(tokenB, "missing", "male")

	seedMatchDirect(foundID, missingID1, 90) // di atas threshold
	seedMatchDirect(foundID, missingID2, 50) // di bawah threshold

	resp := doGet(getMatchRouter(), "/api/v1/matches?min_score=80", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)

	for _, m := range matches {
		match := m.(map[string]any)
		score := int(match["score"].(float64))
		assert.GreaterOrEqual(t, score, 80, "semua match harus punya score >= 80")
	}
}

// 5. Default min_score = 60 (match dengan score < 60 tidak muncul)
func TestGetMatchesDefaultMinScore60(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("defaultscore1@mail.com", "Default Score Satu", "finder")
	tokenB := registerAndLoginMatch("defaultscore2@mail.com", "Default Score Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID := createReportForMatch(tokenB, "missing", "male")
	seedMatchDirect(foundID, missingID, 40) // di bawah default threshold 60

	resp := doGet(getMatchRouter(), "/api/v1/matches", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)

	// Match dengan score 40 tidak boleh muncul (di bawah default 60)
	for _, m := range matches {
		match := m.(map[string]any)
		score := int(match["score"].(float64))
		assert.GreaterOrEqual(t, score, 60, "default filter harus menyaring score < 60")
	}
}

// 6. Hasil diurutkan berdasarkan score DESC
func TestGetMatchesSortedByScoreDesc(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("sortmatch1@mail.com", "Sort Match Satu", "finder")
	tokenB := registerAndLoginMatch("sortmatch2@mail.com", "Sort Match Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID1 := createReportForMatch(tokenB, "missing", "male")
	missingID2 := createReportForMatch(tokenB, "missing", "male")
	missingID3 := createReportForMatch(tokenB, "missing", "male")

	seedMatchDirect(foundID, missingID1, 70)
	seedMatchDirect(foundID, missingID2, 95)
	seedMatchDirect(foundID, missingID3, 80)

	resp := doGet(getMatchRouter(), "/api/v1/matches", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)

	assert.GreaterOrEqual(t, len(matches), 3)

	// Verifikasi urutan score menurun
	for i := 1; i < len(matches); i++ {
		prevScore := int(matches[i-1].(map[string]any)["score"].(float64))
		currScore := int(matches[i].(map[string]any)["score"].(float64))
		assert.GreaterOrEqual(t, prevScore, currScore, "match harus diurutkan score DESC")
	}
}

// 7. Pagination — page dan limit berfungsi
func TestGetMatchesPagination(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("pagmatch1@mail.com", "Pag Match Satu", "finder")
	tokenB := registerAndLoginMatch("pagmatch2@mail.com", "Pag Match Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	for i := 0; i < 5; i++ {
		missingID := createReportForMatch(tokenB, "missing", "male")
		seedMatchDirect(foundID, missingID, 60+i*5)
	}

	resp := doGet(getMatchRouter(), "/api/v1/matches?page=1&limit=2", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)
	meta := data["meta"].(map[string]any)

	assert.LessOrEqual(t, len(matches), 2)
	assert.Equal(t, float64(2), meta["limit"])
	assert.Equal(t, float64(1), meta["page"])
}

// 8. Tidak ada match — response tetap 200 dengan array kosong
func TestGetMatchesEmptyResult(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	token := registerAndLoginMatch("emptymatch1@mail.com", "Empty Match Satu", "finder")

	resp := doGet(getMatchRouter(), "/api/v1/matches", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)
	assert.Empty(t, matches)
}

// 9. Tanpa token → 401
func TestGetMatchesUnauthorized(t *testing.T) {
	resp := doGet(getMatchRouter(), "/api/v1/matches")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 10. min_score tidak valid (< 0) → 400
func TestGetMatchesInvalidMinScore(t *testing.T) {
	truncateTables(testDB)
	token := registerAndLoginMatch("invalidscore1@mail.com", "Invalid Score Satu", "finder")

	resp := doGet(getMatchRouter(), "/api/v1/matches?min_score=-1", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 11. min_score melebihi 100 → 400
func TestGetMatchesMinScoreExceeds100(t *testing.T) {
	truncateTables(testDB)
	token := registerAndLoginMatch("invalidscore2@mail.com", "Invalid Score Dua", "finder")

	resp := doGet(getMatchRouter(), "/api/v1/matches?min_score=101", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 12. limit tidak valid (> 100) → 400
func TestGetMatchesInvalidLimit(t *testing.T) {
	truncateTables(testDB)
	token := registerAndLoginMatch("invalidlimit1@mail.com", "Invalid Limit Satu", "finder")

	resp := doGet(getMatchRouter(), "/api/v1/matches?limit=200", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /matches/:id — GET MATCH BY ID
// ═══════════════════════════════════════════════════════════════════════════════

// 13. Happy path — pemilik found report berhasil lihat match by ID
func TestGetMatchByIDSuccessFoundReporter(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("byid1@mail.com", "ByID Satu", "finder")
	tokenB := registerAndLoginMatch("byid2@mail.com", "ByID Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID := createReportForMatch(tokenB, "missing", "male")
	matchID := seedMatchDirect(foundID, missingID, 88)

	resp := doGet(getMatchRouter(), "/api/v1/matches/"+matchID, tokenA)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, matchID, data["id"])
	assert.NotNil(t, data["score"])
	assert.NotNil(t, data["found_report"])
	assert.NotNil(t, data["missing_report"])
}

// 14. Happy path — pemilik missing report berhasil lihat match by ID
func TestGetMatchByIDSuccessMissingReporter(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("byid3@mail.com", "ByID Tiga", "finder")
	tokenB := registerAndLoginMatch("byid4@mail.com", "ByID Empat", "seeker")

	foundID := createReportForMatch(tokenA, "found", "female")
	missingID := createReportForMatch(tokenB, "missing", "female")
	matchID := seedMatchDirect(foundID, missingID, 77)

	// Token B (pemilik missing report) harus bisa lihat
	resp := doGet(getMatchRouter(), "/api/v1/matches/"+matchID, tokenB)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	assert.Equal(t, matchID, data["id"])
}

// 15. User yang tidak terkait tidak bisa lihat match → 403
func TestGetMatchByIDForbidden(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("forbidden1@mail.com", "Forbidden Satu", "finder")
	tokenB := registerAndLoginMatch("forbidden2@mail.com", "Forbidden Dua", "seeker")
	tokenC := registerAndLoginMatch("forbidden3@mail.com", "Forbidden Tiga", "finder")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID := createReportForMatch(tokenB, "missing", "male")
	matchID := seedMatchDirect(foundID, missingID, 85)

	// Token C tidak terkait dengan match ini
	resp := doGet(getMatchRouter(), "/api/v1/matches/"+matchID, tokenC)
	body := parseBodyReport(resp)

	assert.Equal(t, 403, resp.StatusCode)
	assert.Equal(t, "FORBIDDEN", body["status"])
}

// 16. Match tidak ditemukan → 404
func TestGetMatchByIDNotFound(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	token := registerAndLoginMatch("notfound1@mail.com", "Not Found Satu", "finder")

	resp := doGet(getMatchRouter(), "/api/v1/matches/00000000-0000-0000-0000-000000000000", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 17. ID tidak valid (bukan UUID) → 400
func TestGetMatchByIDInvalidID(t *testing.T) {
	truncateTables(testDB)
	token := registerAndLoginMatch("invalidid1@mail.com", "Invalid ID Satu", "finder")

	resp := doGet(getMatchRouter(), "/api/v1/matches/bukan-uuid", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 18. Tanpa token → 401
func TestGetMatchByIDUnauthorized(t *testing.T) {
	resp := doGet(getMatchRouter(), "/api/v1/matches/00000000-0000-0000-0000-000000000000")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 19. Verifikasi field found_report dan missing_report memiliki sub-field reporter
func TestGetMatchByIDReporterPopulated(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("reporter1@mail.com", "Reporter Satu", "finder")
	tokenB := registerAndLoginMatch("reporter2@mail.com", "Reporter Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID := createReportForMatch(tokenB, "missing", "male")
	matchID := seedMatchDirect(foundID, missingID, 92)

	resp := doGet(getMatchRouter(), "/api/v1/matches/"+matchID, tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)

	foundReport := data["found_report"].(map[string]any)
	assert.NotEmpty(t, foundReport["id"])
	assert.NotNil(t, foundReport["reporter"])

	missingReport := data["missing_report"].(map[string]any)
	assert.NotEmpty(t, missingReport["id"])
	assert.NotNil(t, missingReport["reporter"])
}

// 20. Verifikasi score tersimpan dengan benar
func TestGetMatchByIDScoreAccurate(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("score1@mail.com", "Score Satu", "finder")
	tokenB := registerAndLoginMatch("score2@mail.com", "Score Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	missingID := createReportForMatch(tokenB, "missing", "male")
	expectedScore := 93
	matchID := seedMatchDirect(foundID, missingID, expectedScore)

	resp := doGet(getMatchRouter(), "/api/v1/matches/"+matchID, tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	actualScore := int(data["score"].(float64))
	assert.Equal(t, expectedScore, actualScore)
}

// 21. Verifikasi total_pages terhitung benar
func TestGetMatchesTotalPagesCalculation(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("totalpages1@mail.com", "Total Pages Satu", "finder")
	tokenB := registerAndLoginMatch("totalpages2@mail.com", "Total Pages Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	// Buat 7 match
	for i := 0; i < 7; i++ {
		missingID := createReportForMatch(tokenB, "missing", "male")
		seedMatchDirect(foundID, missingID, 65+i)
	}

	// Limit 3 dari 7 data → harusnya 3 halaman
	resp := doGet(getMatchRouter(), "/api/v1/matches?limit=3", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	meta := data["meta"].(map[string]any)

	total := int(meta["total"].(float64))
	totalPages := int(meta["total_pages"].(float64))

	assert.GreaterOrEqual(t, total, 7)
	assert.Equal(t, 3, totalPages)
}

// 22. User melihat match dari sisi missing reporter (bukan found)
func TestGetMatchesAsMissingReporter(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("missingrole1@mail.com", "Missing Role Satu", "finder")
	tokenB := registerAndLoginMatch("missingrole2@mail.com", "Missing Role Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "female")
	missingID := createReportForMatch(tokenB, "missing", "female")
	seedMatchDirect(foundID, missingID, 78)

	// Token B (pemilik missing report) harus bisa melihat match di GET /matches
	resp := doGet(getMatchRouter(), "/api/v1/matches", tokenB)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	data := body["data"].(map[string]any)
	matches := data["matches"].([]any)
	assert.GreaterOrEqual(t, len(matches), 1)
}

// 23. Token tidak valid (bukan JWT) → 401
func TestGetMatchesInvalidToken(t *testing.T) {
	resp := doGet(getMatchRouter(), "/api/v1/matches", "ini.bukan.token.valid")
	body := parseBodyReport(resp)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 24. Token tidak valid pada GetByID → 401
func TestGetMatchByIDInvalidToken(t *testing.T) {
	resp := doGet(getMatchRouter(), "/api/v1/matches/00000000-0000-0000-0000-000000000000", "ini.bukan.token.valid")
	body := parseBodyReport(resp)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 25. Pagination — page 2 mengembalikan data berikutnya
func TestGetMatchesPage2(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginMatch("page2match1@mail.com", "Page2 Match Satu", "finder")
	tokenB := registerAndLoginMatch("page2match2@mail.com", "Page2 Match Dua", "seeker")

	foundID := createReportForMatch(tokenA, "found", "male")
	var allIDs []string
	for i := 0; i < 4; i++ {
		missingID := createReportForMatch(tokenB, "missing", "male")
		id := seedMatchDirect(foundID, missingID, 65+i*5)
		allIDs = append(allIDs, id)
	}

	respPage1 := doGet(getMatchRouter(), "/api/v1/matches?page=1&limit=2", tokenA)
	bodyPage1 := parseBodyReport(respPage1)
	dataPage1 := bodyPage1["data"].(map[string]any)
	matchesPage1 := dataPage1["matches"].([]any)

	respPage2 := doGet(getMatchRouter(), "/api/v1/matches?page=2&limit=2", tokenA)
	bodyPage2 := parseBodyReport(respPage2)
	dataPage2 := bodyPage2["data"].(map[string]any)
	matchesPage2 := dataPage2["matches"].([]any)

	// ID di page1 tidak boleh sama dengan page2
	idsPage1 := map[string]bool{}
	for _, m := range matchesPage1 {
		idsPage1[m.(map[string]any)["id"].(string)] = true
	}
	for _, m := range matchesPage2 {
		id := m.(map[string]any)["id"].(string)
		assert.False(t, idsPage1[id], "ID di page 2 tidak boleh sama dengan page 1")
	}
}
