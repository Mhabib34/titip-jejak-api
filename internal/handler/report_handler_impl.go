package handler

import (
	"net/http"
	"strconv"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReportHandlerImpl struct {
	usecase usecase.ReportUsecase
}

func NewReportHandlerImpl(usecase usecase.ReportUsecase) *ReportHandlerImpl {
	return &ReportHandlerImpl{usecase}
}

// POST /reports
func (h *ReportHandlerImpl) Create(ctx *gin.Context) {
	reporterID := mustGetUserID(ctx)

	var request dto.CreateReportRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	result, err := h.usecase.Create(ctx, reporterID, &request)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusCreated, dto.WebResponse{
		Status:  "OK",
		Message: "Laporan berhasil dibuat",
		Data:    result,
	})
}

// GET /reports
func (h *ReportHandlerImpl) GetAll(ctx *gin.Context) {
	var query dto.GetReportsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	result, err := h.usecase.GetAll(ctx, query)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status: "OK",
		Data:   result,
	})
}

// GET /reports/my  — harus didaftarkan sebelum /reports/:id di router
func (h *ReportHandlerImpl) GetMyReports(ctx *gin.Context) {
	reporterID := mustGetUserID(ctx)

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	result, err := h.usecase.GetMyReports(ctx, reporterID, page, limit)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status: "OK",
		Data:   result,
	})
}

// GET /reports/:id
func (h *ReportHandlerImpl) GetByID(ctx *gin.Context) {
	id, err := parseUUID(ctx, "id")
	if err != nil {
		exception.ErrorHandler(ctx, exception.NewBadRequestError("id tidak valid"))
		return
	}

	result, err := h.usecase.GetByID(ctx, id)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status: "OK",
		Data:   result,
	})
}

// PUT /reports/:id
func (h *ReportHandlerImpl) Update(ctx *gin.Context) {
	reporterID := mustGetUserID(ctx)

	id, err := parseUUID(ctx, "id")
	if err != nil {
		exception.ErrorHandler(ctx, exception.NewBadRequestError("id tidak valid"))
		return
	}

	var request dto.UpdateReportRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	result, err := h.usecase.Update(ctx, id, reporterID, &request)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status:  "OK",
		Message: "Laporan berhasil diperbarui",
		Data:    result,
	})
}

// DELETE /reports/:id
func (h *ReportHandlerImpl) Delete(ctx *gin.Context) {
	reporterID := mustGetUserID(ctx)

	id, err := parseUUID(ctx, "id")
	if err != nil {
		exception.ErrorHandler(ctx, exception.NewBadRequestError("id tidak valid"))
		return
	}

	if err := h.usecase.Delete(ctx, id, reporterID); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status:  "OK",
		Message: "Laporan berhasil dihapus",
	})
}

// POST /reports/:id/photo
func (h *ReportHandlerImpl) UploadPhoto(ctx *gin.Context) {
	reporterID := mustGetUserID(ctx)

	id, err := parseUUID(ctx, "id")
	if err != nil {
		exception.ErrorHandler(ctx, exception.NewBadRequestError("id tidak valid"))
		return
	}

	file, err := ctx.FormFile("photo")
	if err != nil {
		exception.ErrorHandler(ctx, exception.NewBadRequestError("file foto wajib dikirim"))
		return
	}

	result, err := h.usecase.UploadPhoto(ctx, id, reporterID, file)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status:  "OK",
		Message: "Foto berhasil diupload",
		Data:    result,
	})
}

// GET /map/pins
func (h *ReportHandlerImpl) GetMapPins(ctx *gin.Context) {
	var query dto.GetMapPinsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	result, err := h.usecase.GetMapPins(ctx, query)
	if err != nil {
		exception.ErrorHandler(ctx, err)
		return
	}

	helper.WriteToResponseBody(ctx, http.StatusOK, dto.WebResponse{
		Status: "OK",
		Data:   result,
	})
}

// ── Private helpers ───────────────────────────────────────────────────────────

func mustGetUserID(ctx *gin.Context) uuid.UUID {
	raw, exists := ctx.Get("user_id")
	if !exists {
		panic(exception.NewUnauthorizedError("unauthorized"))
	}
	id, err := uuid.Parse(raw.(interface{ String() string }).String())
	if err != nil {
		panic(exception.NewUnauthorizedError("invalid user id"))
	}
	return id
}

func parseUUID(ctx *gin.Context, param string) (uuid.UUID, error) {
	return uuid.Parse(ctx.Param(param))
}
