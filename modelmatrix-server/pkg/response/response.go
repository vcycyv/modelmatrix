package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents the unified API response format
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Common response codes
const (
	CodeSuccess            = 200
	CodeBadRequest         = 400
	CodeUnauthorized       = 401
	CodeForbidden          = 403
	CodeNotFound           = 404
	CodeConflict           = 409
	CodeInternalError      = 500
	CodeServiceUnavailable = 503
)

// Common response messages
const (
	MsgSuccess            = "success"
	MsgBadRequest         = "bad request"
	MsgUnauthorized       = "unauthorized"
	MsgForbidden          = "forbidden"
	MsgNotFound           = "resource not found"
	MsgConflict           = "resource conflict"
	MsgInternalError      = "internal server error"
	MsgServiceUnavailable = "service unavailable"
)

// Success sends a successful response
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  MsgSuccess,
		Data: data,
	})
}

// SuccessWithMessage sends a successful response with custom message
func SuccessWithMessage(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  msg,
		Data: data,
	})
}

// Created sends a 201 Created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code: CodeSuccess,
		Msg:  "created successfully",
		Data: data,
	})
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Accepted sends a 202 Accepted response
func Accepted(c *gin.Context, data interface{}) {
	c.JSON(http.StatusAccepted, Response{
		Code: CodeSuccess,
		Msg:  "request accepted",
		Data: data,
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgBadRequest
	}
	c.JSON(http.StatusBadRequest, Response{
		Code: CodeBadRequest,
		Msg:  msg,
		Data: nil,
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgUnauthorized
	}
	c.JSON(http.StatusUnauthorized, Response{
		Code: CodeUnauthorized,
		Msg:  msg,
		Data: nil,
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgForbidden
	}
	c.JSON(http.StatusForbidden, Response{
		Code: CodeForbidden,
		Msg:  msg,
		Data: nil,
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgNotFound
	}
	c.JSON(http.StatusNotFound, Response{
		Code: CodeNotFound,
		Msg:  msg,
		Data: nil,
	})
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgConflict
	}
	c.JSON(http.StatusConflict, Response{
		Code: CodeConflict,
		Msg:  msg,
		Data: nil,
	})
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgInternalError
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code: CodeInternalError,
		Msg:  msg,
		Data: nil,
	})
}

// ServiceUnavailable sends a 503 Service Unavailable response
func ServiceUnavailable(c *gin.Context, msg string) {
	if msg == "" {
		msg = MsgServiceUnavailable
	}
	c.JSON(http.StatusServiceUnavailable, Response{
		Code: CodeServiceUnavailable,
		Msg:  msg,
		Data: nil,
	})
}

// Error sends an error response with custom code and message
func Error(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// Paginated represents a paginated response
type Paginated struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// SuccessPaginated sends a paginated response
func SuccessPaginated(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  MsgSuccess,
		Data: Paginated{
			Items:      items,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

