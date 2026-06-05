package exception

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/helper"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ErrorHandler(ctx *gin.Context, err any) {

	if notFoundError(ctx, err) {
		return
	}

	if validationErrors(ctx, err) {
		return
	}

	if uploadImageError(ctx, err) {
		return
	}

	if cancelJobError(ctx, err) {
		return
	}

	if conflictError(ctx, err) {
		return
	}

	if bindingError(ctx, err) {
		return
	}

	if unauthorizedError(ctx, err) {
		return
	}

	if forbiddenError(ctx, err) {
		return
	}

	if badRequestError(ctx, err) {
		return
	}

	internalServerError(ctx, err)
}

func forbiddenError(ctx *gin.Context, err any) bool {
	ex, ok := err.(ForbiddenError)
	if ok {
		webResponse := dto.WebResponse{
			Code:   http.StatusForbidden,
			Status: "FORBIDDEN",
			Error:  ex.Error(),
		}
		helper.WriteToResponseBody(ctx, http.StatusForbidden, webResponse)
		return true
	}
	return false
}

func badRequestError(ctx *gin.Context, err any) bool {
	ex, ok := err.(BadRequestError)
	if ok {
		webResponse := dto.WebResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  ex.Error(),
		}
		helper.WriteToResponseBody(ctx, http.StatusBadRequest, webResponse)
		return true
	}
	return false
}

func unauthorizedError(ctx *gin.Context, err any) bool {
	ex, ok := err.(UnauthorizedError)
	if ok {

		webResponse := dto.WebResponse{
			Code:   http.StatusUnauthorized,
			Status: "UNAUTHORIZED",
			Error:  ex.Error(),
		}

		helper.WriteToResponseBody(ctx, http.StatusUnauthorized, webResponse)
		return true
	}
	return false
}

func bindingError(ctx *gin.Context, err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}

	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError

	if errors.As(e, &syntaxErr) ||
		errors.As(e, &unmarshalTypeErr) ||
		errors.Is(e, io.EOF) ||
		errors.Is(e, io.ErrUnexpectedEOF) {

		webResponse := dto.WebResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  "invalid request body",
		}
		helper.WriteToResponseBody(ctx, http.StatusBadRequest, webResponse)
		return true
	}

	return false
}

func conflictError(ctx *gin.Context, err any) bool {
	ex, ok := err.(ConflictError)
	if ok {
		webResponse := dto.WebResponse{
			Code:   http.StatusConflict,
			Status: "CONFLICT",
			Error:  ex.Error(),
		}
		helper.WriteToResponseBody(ctx, http.StatusConflict, webResponse)
		return true
	}
	return false
}

func validationErrors(ctx *gin.Context, err any) bool {
	ex, ok := err.(validator.ValidationErrors)
	if ok {

		webResponse := dto.WebResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  ex.Error(),
		}

		helper.WriteToResponseBody(ctx, http.StatusBadRequest, webResponse)
		return true
	}
	return false
}

func notFoundError(ctx *gin.Context, err any) bool {
	ex, ok := err.(NotFoundError)
	if ok {

		webResponse := dto.WebResponse{
			Code:   http.StatusNotFound,
			Status: "NOT FOUND",
			Error:  ex.Error(),
		}

		helper.WriteToResponseBody(ctx, http.StatusNotFound, webResponse)
		return true
	}
	return false
}
func uploadImageError(ctx *gin.Context, err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}

	uploadErrors := map[string]bool{
		"file is required":     true,
		"file too large":       true,
		"invalid image format": true,
	}

	if errors.Is(e, http.ErrMissingFile) || uploadErrors[e.Error()] {
		webResponse := dto.WebResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  e.Error(),
		}

		helper.WriteToResponseBody(ctx, http.StatusBadRequest, webResponse)
		return true
	}

	return false
}

func cancelJobError(ctx *gin.Context, err any) bool {
	e, ok := err.(error)
	if !ok {
		return false
	}

	if e.Error() == "job not found or cannot be cancelled" {
		webResponse := dto.WebResponse{
			Code:   http.StatusBadRequest,
			Status: "BAD REQUEST",
			Error:  e.Error(),
		}
		helper.WriteToResponseBody(ctx, http.StatusBadRequest, webResponse)
		return true
	}

	return false
}

func internalServerError(ctx *gin.Context, err any) {

	webResponse := dto.WebResponse{
		Code:   http.StatusInternalServerError,
		Status: "INTERNAL SERVER ERROR",
		Error:  fmt.Sprintf("%v", err),
	}

	helper.WriteToResponseBody(ctx, http.StatusInternalServerError, webResponse)
}
