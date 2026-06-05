package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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

func setupNotificationRouter(db *gorm.DB) *gin.Engine {
	validate := validator.New()

	// User layer
	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo, validate)
	userHandler := handler.NewUserHandlerImpl(userUsecase)

	// Notification layer
	notifRepo := repository.NewNotificationRepository(db)
	notifUsecase := usecase.NewNotificationUsecase(notifRepo)
	notifHandler := handler.NewNotificationHandlerImpl(notifUsecase)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.ErrorRecovery())

	api := r.Group("/api/v1")

	// Auth
	auth := api.Group("/auth")
	auth.POST("/register", userHandler.Create)
	auth.POST("/login", userHandler.Login)

	// Notifications
	// PENTING: read-all harus sebelum /:id/read
	notifs := api.Group("/notifications")
	notifs.Use(middleware.AuthMiddleware())
	notifs.GET("", notifHandler.GetAll)
	notifs.PATCH("/read-all", notifHandler.MarkAllAsRead)
	notifs.PATCH("/:id/read", notifHandler.MarkAsRead)

	return r
}

func truncateNotifications(db *gorm.DB) {
	db.Exec("TRUNCATE TABLE notifications RESTART IDENTITY CASCADE")
}

var notificationRouter *gin.Engine

func getNotificationRouter() *gin.Engine {
	if notificationRouter == nil {
		notificationRouter = setupNotificationRouter(testDB)
	}
	return notificationRouter
}

// registerAndLoginNotif mendaftarkan user dan mengembalikan access token
func registerAndLoginNotif(email, name, role string) string {
	router := getNotificationRouter()
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

// seedNotification menyisipkan notifikasi langsung ke DB
func seedNotification(userID, message string, isRead bool) string {
	var id string
	testDB.Raw(`
		INSERT INTO notifications (user_id, message, is_read)
		VALUES (?, ?, ?)
		RETURNING id
	`, userID, message, isRead).Scan(&id)
	return id
}

// getUserIDByEmail mengambil user ID dari DB berdasarkan email
func getUserIDByEmail(email string) string {
	var id string
	testDB.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&id)
	return id
}

// doPatchNotif mengirimkan PATCH request tanpa body
func doPatchNotif(router *gin.Engine, url, token string) *http.Response {
	req, _ := http.NewRequest(http.MethodPatch, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Result()
}

// ═══════════════════════════════════════════════════════════════════════════════
// GET /notifications — GET ALL NOTIFICATIONS
// ═══════════════════════════════════════════════════════════════════════════════

// 1. Happy path — berhasil mengambil daftar notifikasi
func TestGetNotificationsSuccess(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("notif1@mail.com", "Notif Satu", "finder")
	userID := getUserIDByEmail("notif1@mail.com")
	seedNotification(userID, "Ada kecocokan baru untuk laporan Anda", false)
	seedNotification(userID, "Laporan Anda telah diperbarui", true)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)
	assert.GreaterOrEqual(t, len(notifications), 2)
	assert.NotNil(t, data["unread_count"])
	assert.NotNil(t, data["meta"])
}

// 2. Verifikasi struktur response notifikasi
func TestGetNotificationsResponseStructure(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("notifstruct1@mail.com", "Notif Struct Satu", "finder")
	userID := getUserIDByEmail("notifstruct1@mail.com")
	seedNotification(userID, "Notifikasi test struktur", false)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)
	assert.NotEmpty(t, notifications)

	notif := notifications[0].(map[string]any)
	assert.NotEmpty(t, notif["id"])
	assert.NotEmpty(t, notif["message"])
	assert.NotNil(t, notif["is_read"])
	assert.NotEmpty(t, notif["created_at"])
}

// 3. Filter is_read=false — hanya notifikasi belum dibaca
func TestGetNotificationsFilterUnread(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("filtunread1@mail.com", "Filter Unread Satu", "finder")
	userID := getUserIDByEmail("filtunread1@mail.com")
	seedNotification(userID, "Belum dibaca", false)
	seedNotification(userID, "Sudah dibaca", true)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications?is_read=false", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)

	for _, n := range notifications {
		notif := n.(map[string]any)
		assert.Equal(t, false, notif["is_read"], "hanya notifikasi belum dibaca yang muncul")
	}
}

// 4. Filter is_read=true — hanya notifikasi yang sudah dibaca
func TestGetNotificationsFilterRead(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("filtread1@mail.com", "Filter Read Satu", "seeker")
	userID := getUserIDByEmail("filtread1@mail.com")
	seedNotification(userID, "Notif belum baca", false)
	seedNotification(userID, "Notif sudah baca", true)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications?is_read=true", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)

	for _, n := range notifications {
		notif := n.(map[string]any)
		assert.Equal(t, true, notif["is_read"], "hanya notifikasi sudah dibaca yang muncul")
	}
}

// 5. unread_count mencerminkan jumlah notifikasi belum dibaca
func TestGetNotificationsUnreadCount(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("unreadcount1@mail.com", "Unread Count Satu", "finder")
	userID := getUserIDByEmail("unreadcount1@mail.com")
	seedNotification(userID, "Unread 1", false)
	seedNotification(userID, "Unread 2", false)
	seedNotification(userID, "Read 1", true)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	unreadCount := int(data["unread_count"].(float64))
	assert.Equal(t, 2, unreadCount)
}

// 6. User hanya melihat notifikasinya sendiri
func TestGetNotificationsOwnOnly(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginNotif("ownnotif1@mail.com", "Own Notif Satu", "finder")
	registerAndLoginNotif("ownnotif2@mail.com", "Own Notif Dua", "seeker")

	userAID := getUserIDByEmail("ownnotif1@mail.com")
	userBID := getUserIDByEmail("ownnotif2@mail.com")

	seedNotification(userAID, "Notif untuk user A", false)
	seedNotification(userBID, "Notif untuk user B", false)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", tokenA)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	meta := data["meta"].(map[string]any)
	total := int(meta["total"].(float64))

	// User A hanya boleh lihat 1 notifikasi miliknya
	assert.Equal(t, 1, total)
}

// 7. Tidak ada notifikasi — response 200 dengan array kosong
func TestGetNotificationsEmptyResult(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("emptynotif1@mail.com", "Empty Notif Satu", "finder")

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)
	assert.Empty(t, notifications)
	assert.Equal(t, float64(0), data["unread_count"])
}

// 8. Pagination berfungsi dengan benar
func TestGetNotificationsPagination(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("pagnotif1@mail.com", "Pag Notif Satu", "finder")
	userID := getUserIDByEmail("pagnotif1@mail.com")

	for i := 0; i < 5; i++ {
		seedNotification(userID, fmt.Sprintf("Notifikasi ke-%d", i+1), false)
	}

	resp := doGet(getNotificationRouter(), "/api/v1/notifications?page=1&limit=2", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)
	meta := data["meta"].(map[string]any)

	assert.LessOrEqual(t, len(notifications), 2)
	assert.Equal(t, float64(2), meta["limit"])
	assert.Equal(t, float64(1), meta["page"])
}

// 9. Notifikasi diurutkan terbaru dulu (created_at DESC)
func TestGetNotificationsSortedByCreatedAtDesc(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("sortnotif1@mail.com", "Sort Notif Satu", "finder")
	userID := getUserIDByEmail("sortnotif1@mail.com")

	seedNotification(userID, "Notif lama", false)
	seedNotification(userID, "Notif terbaru", false)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)
	assert.GreaterOrEqual(t, len(notifications), 2)

	// Verifikasi created_at notif pertama >= kedua (DESC)
	first := notifications[0].(map[string]any)["created_at"].(string)
	second := notifications[1].(map[string]any)["created_at"].(string)
	assert.GreaterOrEqual(t, first, second, "notifikasi harus diurutkan created_at DESC")
}

// 10. Tanpa token → 401
func TestGetNotificationsUnauthorized(t *testing.T) {
	resp := doGet(getNotificationRouter(), "/api/v1/notifications")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 11. Token tidak valid → 401
func TestGetNotificationsInvalidToken(t *testing.T) {
	resp := doGet(getNotificationRouter(), "/api/v1/notifications", "ini.bukan.token.valid")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 12. limit tidak valid (> 100) → 400
func TestGetNotificationsInvalidLimit(t *testing.T) {
	truncateTables(testDB)
	token := registerAndLoginNotif("invalidlimitnotif1@mail.com", "Invalid Limit Notif", "finder")

	resp := doGet(getNotificationRouter(), "/api/v1/notifications?limit=200", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// PATCH /notifications/:id/read — MARK AS READ
// ═══════════════════════════════════════════════════════════════════════════════

// 13. Happy path — menandai notifikasi sebagai sudah dibaca
func TestMarkNotificationAsReadSuccess(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("markread1@mail.com", "Mark Read Satu", "finder")
	userID := getUserIDByEmail("markread1@mail.com")
	notifID := seedNotification(userID, "Notifikasi belum dibaca", false)

	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/"+notifID+"/read", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.NotNil(t, body["message"])
}

// 14. Setelah mark as read, is_read menjadi true di DB
func TestMarkNotificationAsReadPersisted(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("markreadpersist1@mail.com", "Mark Read Persist Satu", "finder")
	userID := getUserIDByEmail("markreadpersist1@mail.com")
	notifID := seedNotification(userID, "Notifikasi akan dibaca", false)

	doPatchNotif(getNotificationRouter(), "/api/v1/notifications/"+notifID+"/read", token)

	// Verifikasi di DB
	var isRead bool
	testDB.Raw("SELECT is_read FROM notifications WHERE id = ?", notifID).Scan(&isRead)
	assert.True(t, isRead, "is_read harus true setelah mark as read")
}

// 15. Mark as read notifikasi milik user lain → 404
func TestMarkNotificationAsReadForbidden(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	registerAndLoginNotif("markforbid1@mail.com", "Mark Forbid Satu", "finder")
	tokenB := registerAndLoginNotif("markforbid2@mail.com", "Mark Forbid Dua", "seeker")

	userAID := getUserIDByEmail("markforbid1@mail.com")
	notifID := seedNotification(userAID, "Notif milik user A", false)

	// User B mencoba mark notifikasi user A
	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/"+notifID+"/read", tokenB)
	body := parseBodyReport(resp)

	// Repository menggunakan gorm.ErrRecordNotFound untuk isolasi data per user → 404
	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 16. Notifikasi tidak ditemukan → 404
func TestMarkNotificationAsReadNotFound(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("marknotfound1@mail.com", "Mark Not Found Satu", "finder")

	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/00000000-0000-0000-0000-000000000000/read", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, "NOT FOUND", body["status"])
}

// 17. ID tidak valid (bukan UUID) → 400
func TestMarkNotificationAsReadInvalidID(t *testing.T) {
	truncateTables(testDB)
	token := registerAndLoginNotif("markbadid1@mail.com", "Mark Bad ID Satu", "finder")

	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/bukan-uuid/read", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, "BAD REQUEST", body["status"])
}

// 18. Tanpa token → 401
func TestMarkNotificationAsReadUnauthorized(t *testing.T) {
	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/00000000-0000-0000-0000-000000000000/read", "")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 19. Mark notifikasi yang sudah dibaca — idempoten (tetap 200)
func TestMarkNotificationAsReadIdempotent(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("idemread1@mail.com", "Idem Read Satu", "finder")
	userID := getUserIDByEmail("idemread1@mail.com")
	notifID := seedNotification(userID, "Sudah dibaca sebelumnya", true)

	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/"+notifID+"/read", token)
	body := parseBodyReport(resp)

	// Harus tetap 200 meski sudah dibaca
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// PATCH /notifications/read-all — MARK ALL AS READ
// ═══════════════════════════════════════════════════════════════════════════════

// 20. Happy path — semua notifikasi ditandai sudah dibaca
func TestMarkAllNotificationsAsReadSuccess(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("readall1@mail.com", "Read All Satu", "finder")
	userID := getUserIDByEmail("readall1@mail.com")
	seedNotification(userID, "Notif 1", false)
	seedNotification(userID, "Notif 2", false)
	seedNotification(userID, "Notif 3", false)

	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
	assert.NotNil(t, body["message"])
}

// 21. Setelah mark-all-read, semua notifikasi user is_read=true
func TestMarkAllNotificationsAsReadPersisted(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("readallpersist1@mail.com", "Read All Persist Satu", "finder")
	userID := getUserIDByEmail("readallpersist1@mail.com")
	seedNotification(userID, "Notif A", false)
	seedNotification(userID, "Notif B", false)

	doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", token)

	// Verifikasi semua notif user sudah is_read=true
	var unreadCount int64
	testDB.Raw("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = false", userID).Scan(&unreadCount)
	assert.Equal(t, int64(0), unreadCount, "tidak boleh ada notifikasi belum dibaca setelah mark-all-read")
}

// 22. Mark-all-read hanya mempengaruhi notifikasi user yang sedang login (bukan user lain)
func TestMarkAllNotificationsAsReadIsolated(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	tokenA := registerAndLoginNotif("readalliso1@mail.com", "Read All Iso Satu", "finder")
	registerAndLoginNotif("readalliso2@mail.com", "Read All Iso Dua", "seeker")

	userAID := getUserIDByEmail("readalliso1@mail.com")
	userBID := getUserIDByEmail("readalliso2@mail.com")

	seedNotification(userAID, "Notif user A", false)
	seedNotification(userBID, "Notif user B — tidak boleh terpengaruh", false)

	// User A mark all read
	doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", tokenA)

	// Notifikasi user B harus tetap belum dibaca
	var unreadCountB int64
	testDB.Raw("SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = false", userBID).Scan(&unreadCountB)
	assert.Equal(t, int64(1), unreadCountB, "notifikasi user B tidak boleh terpengaruh")
}

// 23. Mark-all-read saat tidak ada notifikasi — tetap 200 (idempoten)
func TestMarkAllNotificationsAsReadEmpty(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("readallempty1@mail.com", "Read All Empty Satu", "finder")

	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", token)
	body := parseBodyReport(resp)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "OK", body["status"])
}

// 24. Tanpa token → 401
func TestMarkAllNotificationsAsReadUnauthorized(t *testing.T) {
	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", "")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 25. unread_count menjadi 0 setelah mark-all-read
func TestGetNotificationsUnreadCountAfterMarkAll(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("unreadafter1@mail.com", "Unread After Satu", "finder")
	userID := getUserIDByEmail("unreadafter1@mail.com")
	seedNotification(userID, "Unread A", false)
	seedNotification(userID, "Unread B", false)

	// Sebelum mark-all
	respBefore := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	bodyBefore := parseBodyReport(respBefore)
	dataBefore := bodyBefore["data"].(map[string]any)
	assert.Equal(t, float64(2), dataBefore["unread_count"])

	// Mark all read
	doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", token)

	// Sesudah mark-all
	respAfter := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	bodyAfter := parseBodyReport(respAfter)
	dataAfter := bodyAfter["data"].(map[string]any)
	assert.Equal(t, float64(0), dataAfter["unread_count"])
}

// 26. Token tidak valid pada mark-all-read → 401
func TestMarkAllNotificationsAsReadInvalidToken(t *testing.T) {
	resp := doPatchNotif(getNotificationRouter(), "/api/v1/notifications/read-all", "ini.bukan.token.valid")
	body := parseBodyReport(resp)

	assert.Equal(t, 401, resp.StatusCode)
	assert.Equal(t, "UNAUTHORIZED", body["status"])
}

// 27. Verifikasi total_pages pada pagination notifikasi
func TestGetNotificationsTotalPages(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("totalpagenotif1@mail.com", "Total Page Notif Satu", "finder")
	userID := getUserIDByEmail("totalpagenotif1@mail.com")

	for i := 0; i < 7; i++ {
		seedNotification(userID, fmt.Sprintf("Notifikasi %d", i+1), false)
	}

	resp := doGet(getNotificationRouter(), "/api/v1/notifications?limit=3", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	meta := data["meta"].(map[string]any)
	totalPages := int(meta["total_pages"].(float64))
	total := int(meta["total"].(float64))

	assert.GreaterOrEqual(t, total, 7)
	assert.Equal(t, 3, totalPages) // ceil(7/3) = 3
}

// 28. Notifikasi dengan report_id dan match_id tersimpan dengan benar (opsional field)
func TestGetNotificationsWithOptionalFields(t *testing.T) {
	truncateNotifications(testDB)
	truncateTables(testDB)

	token := registerAndLoginNotif("optfield1@mail.com", "Opt Field Satu", "finder")
	userID := getUserIDByEmail("optfield1@mail.com")

	// Notif tanpa report_id dan match_id (field opsional)
	seedNotification(userID, "Notifikasi tanpa report/match", false)

	resp := doGet(getNotificationRouter(), "/api/v1/notifications", token)
	body := parseBodyReport(resp)

	data := body["data"].(map[string]any)
	notifications := data["notifications"].([]any)
	assert.NotEmpty(t, notifications)

	notif := notifications[0].(map[string]any)
	// report_id dan match_id boleh nil
	_ = notif["report_id"] // bisa nil, tidak error
	_ = notif["match_id"]  // bisa nil, tidak error
	assert.NotEmpty(t, notif["id"])
	assert.NotEmpty(t, notif["message"])
}
