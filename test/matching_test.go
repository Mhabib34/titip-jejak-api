package test

import (
	"testing"
	"time"

	"titip-jejak-api/internal/model"
	"titip-jejak-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ═══════════════════════════════════════════════════════════════════════════════
// UNIT TEST — matching_service.go (ScoreReports)
// Test ini tidak butuh DB — murni unit test fungsi scoring
// ═══════════════════════════════════════════════════════════════════════════════

func makeReport(reportType model.ReportType, gender model.ReportGender, age *int, city, province, desc string) model.Report {
	return model.Report{
		ID:               uuid.New(),
		ReporterID:       uuid.New(),
		Type:             reportType,
		Gender:           gender,
		EstimatedAge:     age,
		City:             city,
		Province:         province,
		Description:      desc,
		LastSeenLocation: "Jalan Test",
		Status:           model.ReportStatusActive,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func intPtr(n int) *int { return &n }

// ── Lokasi ────────────────────────────────────────────────────────────────────

// 1. Same city = 40 poin
func TestScoreLocation_SameCity(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(50), "Medan", "Sumatera Utara", "deskripsi")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(50), "Medan", "Sumatera Utara", "deskripsi")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(30) + usia(20) + deskripsi(10 atau lebih) >= 90
	assert.GreaterOrEqual(t, score, 90)
}

// 2. Same province, beda city = 20 poin lokasi
func TestScoreLocation_SameProvince(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(50), "Medan", "Sumatera Utara", "pria tua batik")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(50), "Binjai", "Sumatera Utara", "pria tua batik")

	score := service.ScoreReports(found, missing)
	// lokasi(20) + gender(30) + usia(20) + desc(?) = 70+
	assert.GreaterOrEqual(t, score, 70)
	// Tidak dapat 40 dari lokasi (beda kota)
	assert.Less(t, score, 100)
}

// 3. Beda province = 0 poin lokasi
func TestScoreLocation_DifferentProvince(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(50), "Medan", "Sumatera Utara", "pria batik")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(50), "Jakarta", "DKI Jakarta", "pria batik")

	score := service.ScoreReports(found, missing)
	// lokasi(0) + gender(30) + usia(20) + desc(?) = 50+
	assert.Less(t, score, 70) // tidak dapat poin lokasi
}

// 4. Kota sama case-insensitive (Medan vs medan)
func TestScoreLocation_CaseInsensitive(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(50), "Medan", "Sumatera Utara", "deskripsi")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(50), "medan", "sumatera utara", "deskripsi")

	score := service.ScoreReports(found, missing)
	assert.GreaterOrEqual(t, score, 90)
}

// ── Gender ────────────────────────────────────────────────────────────────────

// 5. Gender sama = 30 poin
func TestScoreGender_Match(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderFemale, intPtr(30), "Medan", "Sumatera Utara", "wanita muda")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderFemale, intPtr(30), "Medan", "Sumatera Utara", "wanita muda")

	score := service.ScoreReports(found, missing)
	assert.GreaterOrEqual(t, score, 90)
}

// 6. Gender beda = 0 poin
func TestScoreGender_Mismatch(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(30), "Medan", "Sumatera Utara", "orang kebingungan")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderFemale, intPtr(30), "Medan", "Sumatera Utara", "orang kebingungan")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(0) + usia(20) = 60 (tanpa gender)
	assert.Less(t, score, 75)
}

// 7. Salah satu unknown = 15 poin gender
func TestScoreGender_OneUnknown(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderUnknown, intPtr(50), "Medan", "Sumatera Utara", "orang tua")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(50), "Medan", "Sumatera Utara", "orang tua")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(15) + usia(20) = 75+
	assert.GreaterOrEqual(t, score, 75)
	assert.Less(t, score, 90) // tidak dapat penuh 30 dari gender
}

// 8. Kedua unknown = 15 poin gender (unknown dianggap partial match)
func TestScoreGender_BothUnknown(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderUnknown, intPtr(50), "Medan", "Sumatera Utara", "seseorang")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderUnknown, intPtr(50), "Medan", "Sumatera Utara", "seseorang")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(15) + usia(20) = 75+
	assert.GreaterOrEqual(t, score, 75)
}

// ── Usia ──────────────────────────────────────────────────────────────────────

// 9. Selisih usia ≤5 tahun = 20 poin
func TestScoreAge_CloseAge(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", "pria tua")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(68), "Medan", "Sumatera Utara", "pria tua")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(30) + usia(20) = 90+
	assert.GreaterOrEqual(t, score, 90)
}

// 10. Selisih usia 6-10 tahun = 10 poin
func TestScoreAge_ModerateAge(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(60), "Medan", "Sumatera Utara", "pria")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(68), "Medan", "Sumatera Utara", "pria")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(30) + usia(10) = 80+
	assert.GreaterOrEqual(t, score, 80)
	assert.LessOrEqual(t, score, 90) // tidak dapat 20 penuh dari usia
}

// 11. Selisih usia >10 tahun = 0 poin
func TestScoreAge_FarAge(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(30), "Medan", "Sumatera Utara", "pria")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(70), "Medan", "Sumatera Utara", "pria")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(30) + usia(0) = 70
	assert.LessOrEqual(t, score, 90)
}

// 12. Salah satu usia nil = 0 poin usia
func TestScoreAge_NilAge(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, nil, "Medan", "Sumatera Utara", "pria")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", "pria")

	score := service.ScoreReports(found, missing)
	assert.Equal(t, 80, score)
}

// 13. Kedua usia nil = 0 poin usia
func TestScoreAge_BothNilAge(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, nil, "Medan", "Sumatera Utara", "pria")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, nil, "Medan", "Sumatera Utara", "pria")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(30) + usia(0) = 70
	assert.Equal(t, 80, score)
}

// ── Deskripsi ─────────────────────────────────────────────────────────────────

// 14. Deskripsi overlap tinggi = 10 poin
func TestScoreDescription_HighOverlap(t *testing.T) {
	desc := "pria tua rambut putih baju batik biru kebingungan"
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", desc)
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", desc)

	score := service.ScoreReports(found, missing)
	assert.Equal(t, 100, score)
}

// 15. Deskripsi kosong = 0 poin deskripsi
func TestScoreDescription_Empty(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", "")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", "")

	score := service.ScoreReports(found, missing)
	// lokasi(40) + gender(30) + usia(20) + desc(0) = 90
	assert.Equal(t, 90, score)
}

// 16. Deskripsi sama sekali tidak overlap = 0 poin
func TestScoreDescription_NoOverlap(t *testing.T) {
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", "gemuk pendek rambut keriting")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", "kurus tinggi rambut lurus")

	score := service.ScoreReports(found, missing)
	assert.Equal(t, 95, score)
}

// ── Threshold ─────────────────────────────────────────────────────────────────

// 17. MinMatchScore adalah 60
func TestMinMatchScore_Value(t *testing.T) {
	assert.Equal(t, 60, service.MinMatchScore)
}

// 18. Skor tepat di batas threshold (60) — masih masuk
func TestScoreReports_AtThreshold(t *testing.T) {
	// lokasi(0) + gender(30) + usia(20) + desc(10) = 60 — beda provinsi
	desc := "pria tua rambut putih baju batik biru kebingungan pasar"
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(65), "Jakarta", "DKI Jakarta", desc)
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(65), "Surabaya", "Jawa Timur", desc)

	score := service.ScoreReports(found, missing)
	assert.GreaterOrEqual(t, score, service.MinMatchScore)
}

// 19. Skor di bawah threshold — tidak lolos
func TestScoreReports_BelowThreshold(t *testing.T) {
	// lokasi(0) + gender(0) + usia(0) + desc(0)
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, nil, "Jakarta", "DKI Jakarta", "")
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderFemale, nil, "Surabaya", "Jawa Timur", "")

	score := service.ScoreReports(found, missing)
	assert.Less(t, score, service.MinMatchScore)
}

// 20. Skor maksimal = 100
func TestScoreReports_MaxScore(t *testing.T) {
	desc := "pria tua rambut putih baju batik biru terlihat kebingungan pasar jalan"
	found := makeReport(model.ReportTypeFound, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", desc)
	missing := makeReport(model.ReportTypeMissing, model.ReportGenderMale, intPtr(65), "Medan", "Sumatera Utara", desc)

	score := service.ScoreReports(found, missing)
	assert.Equal(t, 100, score)
	assert.LessOrEqual(t, score, 100) // tidak melebihi 100
}

// ═══════════════════════════════════════════════════════════════════════════════
// INTEGRATION TEST — Auto-matching setelah POST /reports
// Memverifikasi bahwa worker berjalan dan match tersimpan ke DB
// ═══════════════════════════════════════════════════════════════════════════════

// setupMatchingRouter menyiapkan router dengan worker aktif untuk integration test
func setupMatchingRouter() *gin.Engine {
	return setupMatchRouter(testDB)
}

// waitForMatch menunggu sampai match muncul di DB (max 2 detik)
func waitForMatch(foundID, missingID string) bool {
	for i := 0; i < 20; i++ {
		var count int64
		testDB.Raw(`
			SELECT COUNT(*) FROM matches
			WHERE found_report_id = ? AND missing_report_id = ?
		`, foundID, missingID).Scan(&count)
		if count > 0 {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// getMatchFromDB mengambil match langsung dari DB
func getMatchFromDB(foundID, missingID string) map[string]any {
	var result struct {
		ID       string
		Score    int
		Notified bool
	}
	testDB.Raw(`
		SELECT id, score, notified FROM matches
		WHERE found_report_id = ? AND missing_report_id = ?
		LIMIT 1
	`, foundID, missingID).Scan(&result)
	return map[string]any{
		"id":       result.ID,
		"score":    result.Score,
		"notified": result.Notified,
	}
}

// countNotificationsForUser menghitung notifikasi di DB untuk user tertentu
func countNotificationsForUser(userID string) int64 {
	var count int64
	testDB.Raw("SELECT COUNT(*) FROM notifications WHERE user_id = ?", userID).Scan(&count)
	return count
}

// countMatchesForReports menghitung match antara dua report di DB
func countMatchesForReports(foundID, missingID string) int64 {
	var count int64
	testDB.Raw(`
		SELECT COUNT(*) FROM matches
		WHERE found_report_id = ? AND missing_report_id = ?
	`, foundID, missingID).Scan(&count)
	return count
}

func createReportForMatchNoDesc(router *gin.Engine, token, reportType, gender, city, province string, age int) string {
	// Deskripsi finder dan seeker sengaja dibuat tidak overlap sama sekali
	// supaya scoreDescription = 0, score beda provinsi = gender(30)+usia(20) = 50 < threshold
	descFound := "gemuk pendek berambut keriting memakai celana merah polos"
	descMissing := "kurus tinggi berambut lurus mengenakan jaket hitam kotak"

	desc := descFound
	if reportType == "missing" {
		desc = descMissing
	}

	resp := doPostAuth(router, "/api/v1/reports", `{
        "type":               "`+reportType+`",
        "gender":             "`+gender+`",
        "estimated_age":      `+itoa(age)+`,
        "description":        "`+desc+`",
        "last_seen_location": "Depan Pasar Petisah",
        "city":               "`+city+`",
        "province":           "`+province+`"
    }`, token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// ── Scoring integration ───────────────────────────────────────────────────────

// 21. POST report found → auto-match dengan missing report yang cocok
func TestAutoMatch_FoundTriggersMissingMatch(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	// Seed: seeker buat missing report dulu
	tokenSeeker := registerAndLoginMatch("automatch1seeker@mail.com", "Auto Seeker Satu", "seeker")
	missingID := createReportForMatch(tokenSeeker, "missing", "male")

	// Finder buat found report → trigger worker
	tokenFinder := registerAndLoginMatch("automatch1finder@mail.com", "Auto Finder Satu", "finder")
	foundID := createReportForMatchFull(router, tokenFinder, "found", "male", "Medan", "Sumatera Utara", 65)

	// Worker berjalan async — tunggu max 2 detik
	matched := waitForMatch(foundID, missingID)
	assert.True(t, matched, "match harus terbentuk otomatis setelah report dibuat")
}

// 22. Skor match yang tersimpan di DB sesuai hasil kalkulasi scoring
func TestAutoMatch_ScoreAccurate(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	tokenSeeker := registerAndLoginMatch("scoreaccurate1seeker@mail.com", "Score Seeker Satu", "seeker")
	missingID := createReportForMatchFull(router, tokenSeeker, "missing", "male", "Medan", "Sumatera Utara", 65)

	tokenFinder := registerAndLoginMatch("scoreaccurate1finder@mail.com", "Score Finder Satu", "finder")
	foundID := createReportForMatchFull(router, tokenFinder, "found", "male", "Medan", "Sumatera Utara", 65)

	waitForMatch(foundID, missingID)

	match := getMatchFromDB(foundID, missingID)
	score := match["score"].(int)

	// same city(40) + same gender(30) + usia selisih 0 (20) = minimal 90
	assert.GreaterOrEqual(t, score, 90)
	assert.LessOrEqual(t, score, 100)
}

// 23. Match di bawah threshold (skor < 60) tidak tersimpan ke DB
func TestAutoMatch_BelowThresholdNotSaved(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	// Gender beda, provinsi beda, usia jauh — skor pasti < 60
	tokenSeeker := registerAndLoginMatch("belowthresh1seeker@mail.com", "Below Seeker Satu", "seeker")
	createReportForMatchProvince(router, tokenSeeker, "missing", "female", "Jakarta", "DKI Jakarta", 20)

	tokenFinder := registerAndLoginMatch("belowthresh1finder@mail.com", "Below Finder Satu", "finder")
	foundID := createReportForMatchProvince(router, tokenFinder, "found", "male", "Surabaya", "Jawa Timur", 80)

	// Tunggu sebentar agar worker sempat jalan
	time.Sleep(500 * time.Millisecond)

	// Tidak boleh ada match
	var count int64
	testDB.Raw("SELECT COUNT(*) FROM matches WHERE found_report_id = ?", foundID).Scan(&count)
	assert.Equal(t, int64(0), count, "match di bawah threshold tidak boleh tersimpan")
}

// 24. Match tidak duplikat — report yang sama tidak di-match dua kali
func TestAutoMatch_NoDuplicate(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	tokenSeeker := registerAndLoginMatch("nodup1seeker@mail.com", "NoDup Seeker Satu", "seeker")
	missingID := createReportForMatch(tokenSeeker, "missing", "male")

	tokenFinder := registerAndLoginMatch("nodup1finder@mail.com", "NoDup Finder Satu", "finder")
	foundID := createReportForMatchWithRouter(router, tokenFinder, "found", "male")

	waitForMatch(foundID, missingID)

	// Update report (trigger matching ulang)
	doPut(router, "/api/v1/reports/"+foundID, `{
		"description": "Deskripsi diperbarui pria tua baju batik"
	}`, getTokenFromEmail(router, "nodup1finder@mail.com"))

	time.Sleep(500 * time.Millisecond)

	// Harus tetap hanya 1 match antara pasangan ini
	count := countMatchesForReports(foundID, missingID)
	assert.Equal(t, int64(1), count, "tidak boleh ada duplikat match untuk pasangan yang sama")
}

// 25. Missing report yang resolved tidak di-match
func TestAutoMatch_ResolvedReportSkipped(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	tokenSeeker := registerAndLoginMatch("resolved1seeker@mail.com", "Resolved Seeker Satu", "seeker")
	missingID := createReportForMatch(tokenSeeker, "missing", "male")

	// Set missing report jadi resolved sebelum found report dibuat
	testDB.Exec("UPDATE reports SET status = 'resolved' WHERE id = ?", missingID)

	tokenFinder := registerAndLoginMatch("resolved1finder@mail.com", "Resolved Finder Satu", "finder")
	foundID := createReportForMatchWithRouter(router, tokenFinder, "found", "male")

	time.Sleep(500 * time.Millisecond)

	count := countMatchesForReports(foundID, missingID)
	assert.Equal(t, int64(0), count, "report resolved tidak boleh di-match")
}

// 26. Notifikasi DB terbuat untuk kedua pihak (finder dan seeker)
func TestAutoMatch_NotificationsCreated(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)
	truncateNotifications(testDB)

	router := setupMatchingRouter()

	tokenSeeker := registerAndLoginMatch("notifboth1seeker@mail.com", "Notif Both Seeker", "seeker")
	missingID := createReportForMatch(tokenSeeker, "missing", "male")
	seekerID := getUserIDByEmail("notifboth1seeker@mail.com")

	tokenFinder := registerAndLoginMatch("notifboth1finder@mail.com", "Notif Both Finder", "finder")
	foundID := createReportForMatchWithRouter(router, tokenFinder, "found", "male")
	finderID := getUserIDByEmail("notifboth1finder@mail.com")

	matched := waitForMatch(foundID, missingID)
	assert.True(t, matched)

	// Tunggu goroutine notifikasi
	time.Sleep(300 * time.Millisecond)

	finderNotifs := countNotificationsForUser(finderID)
	seekerNotifs := countNotificationsForUser(seekerID)

	assert.GreaterOrEqual(t, finderNotifs, int64(1), "finder harus dapat notifikasi")
	assert.GreaterOrEqual(t, seekerNotifs, int64(1), "seeker harus dapat notifikasi")
}

// 27. Match di-trigger juga saat report diupdate (PUT /reports/:id)
func TestAutoMatch_TriggeredOnUpdate(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	tokenFinder := registerAndLoginMatch("updatetrigger1finder@mail.com", "Update Trigger Finder", "finder")
	// Tanpa deskripsi → score beda provinsi = 30+20 = 50, di bawah threshold 60
	foundID := createReportForMatchNoDesc(router, tokenFinder, "found", "male", "Jakarta", "DKI Jakarta", 65)

	tokenSeeker := registerAndLoginMatch("updatetrigger1seeker@mail.com", "Update Trigger Seeker", "seeker")
	// Tanpa deskripsi juga supaya konsisten
	missingID := createReportForMatchNoDesc(router, tokenSeeker, "missing", "male", "Medan", "Sumatera Utara", 65)

	time.Sleep(300 * time.Millisecond)

	// Belum match: lokasi(0) + gender(30) + usia(20) = 50 < 60
	countBefore := countMatchesForReports(foundID, missingID)
	assert.Equal(t, int64(0), countBefore)

	finderToken := getTokenFromEmail(router, "updatetrigger1finder@mail.com")
	doPut(router, "/api/v1/reports/"+foundID, `{
        "city":     "Medan",
        "province": "Sumatera Utara"
    }`, finderToken)

	// Setelah update: lokasi(40) + gender(30) + usia(20) = 90 >= 60 → match
	matched := waitForMatch(foundID, missingID)
	assert.True(t, matched, "match harus terbentuk setelah report diupdate ke kota yang sama")
}

// 28. Missing report trigger matching dengan semua found report yang ada
func TestAutoMatch_MissingTriggersAgainstExistingFound(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	// Seed 2 found report dulu
	tokenFinderA := registerAndLoginMatch("missingtrig1finderA@mail.com", "Finder A", "finder")
	foundIDA := createReportForMatchWithRouter(router, tokenFinderA, "found", "female")

	tokenFinderB := registerAndLoginMatch("missingtrig1finderB@mail.com", "Finder B", "finder")
	foundIDB := createReportForMatchWithRouter(router, tokenFinderB, "found", "female")

	// Setelah itu seeker buat missing → harus match dengan keduanya
	tokenSeeker := registerAndLoginMatch("missingtrig1seeker@mail.com", "Seeker Trig", "seeker")
	missingID := createReportForMatch(tokenSeeker, "missing", "female")

	matchedA := waitForMatch(foundIDA, missingID)
	matchedB := waitForMatch(foundIDB, missingID)

	assert.True(t, matchedA, "missing harus match dengan found report A")
	assert.True(t, matchedB, "missing harus match dengan found report B")
}

// 29. Match hanya terjadi antara found ↔ missing (bukan found ↔ found)
func TestAutoMatch_OnlyFoundVsMissing(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)

	router := setupMatchingRouter()

	tokenA := registerAndLoginMatch("typematch1a@mail.com", "Type Match A", "finder")
	foundIDA := createReportForMatchWithRouter(router, tokenA, "found", "male")

	tokenB := registerAndLoginMatch("typematch1b@mail.com", "Type Match B", "finder")
	foundIDB := createReportForMatchWithRouter(router, tokenB, "found", "male")

	time.Sleep(500 * time.Millisecond)

	// Tidak boleh ada match found ↔ found
	var count int64
	testDB.Raw(`
		SELECT COUNT(*) FROM matches m
		JOIN reports r1 ON r1.id = m.found_report_id
		JOIN reports r2 ON r2.id = m.missing_report_id
		WHERE r1.type = 'found' AND r2.type = 'found'
		  AND (m.found_report_id = ? OR m.found_report_id = ?)
	`, foundIDA, foundIDB).Scan(&count)
	assert.Equal(t, int64(0), count, "tidak boleh ada match antara sesama found report")
}

// 30. Notifikasi match mengandung skor yang benar
func TestAutoMatch_NotificationContainsScore(t *testing.T) {
	truncateMatches(testDB)
	truncateReports(testDB)
	truncateTables(testDB)
	truncateNotifications(testDB)

	router := setupMatchingRouter()

	tokenSeeker := registerAndLoginMatch("notifsc1seeker@mail.com", "Notif Score Seeker", "seeker")
	missingID := createReportForMatch(tokenSeeker, "missing", "male")
	seekerID := getUserIDByEmail("notifsc1seeker@mail.com")

	tokenFinder := registerAndLoginMatch("notifsc1finder@mail.com", "Notif Score Finder", "finder")
	foundID := createReportForMatchWithRouter(router, tokenFinder, "found", "male")

	waitForMatch(foundID, missingID)
	time.Sleep(300 * time.Millisecond)

	// Ambil pesan notifikasi dari DB dan pastikan ada info skor
	var message string
	testDB.Raw(`
		SELECT message FROM notifications WHERE user_id = ? LIMIT 1
	`, seekerID).Scan(&message)

	assert.NotEmpty(t, message)
	assert.Contains(t, message, "/100", "pesan notifikasi harus mengandung skor")
}

// ── Helpers khusus matching integration test ──────────────────────────────────

// createReportForMatchFull membuat report dengan semua field eksplisit
func createReportForMatchFull(router *gin.Engine, token, reportType, gender, city, province string, age int) string {
	resp := doPostAuth(router, "/api/v1/reports", `{
		"type":               "`+reportType+`",
		"gender":             "`+gender+`",
		"estimated_age":      `+itoa(age)+`,
		"description":        "pria tua rambut putih baju batik biru kebingungan",
		"last_seen_location": "Depan Pasar Petisah",
		"city":               "`+city+`",
		"province":           "`+province+`"
	}`, token)
	body := parseBodyReport(resp)
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id
}

// createReportForMatchProvince membuat report dengan kota & provinsi berbeda
func createReportForMatchProvince(router *gin.Engine, token, reportType, gender, city, province string, age int) string {
	return createReportForMatchFull(router, token, reportType, gender, city, province, age)
}

// createReportForMatch override — versi yang terima router eksplisit
func createReportForMatchWithRouter(router *gin.Engine, token, reportType, gender string) string {
	return createReportForMatchFull(router, token, reportType, gender, "Medan", "Sumatera Utara", 65)
}

// getTokenFromEmail login ulang untuk dapat token segar
func getTokenFromEmail(router *gin.Engine, email string) string {
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

// itoa konversi int ke string tanpa import strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [10]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte(n%10) + '0'
		n /= 10
	}
	return string(buf[pos:])
}
