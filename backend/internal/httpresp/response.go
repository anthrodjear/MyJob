package httpresp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody holds the error details.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Created sends a 201 JSON response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// OK sends a 200 JSON response.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// BadRequest sends a 400 JSON response with a machine-readable code.
func BadRequest(c *gin.Context, code string, msg string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{Code: code, Message: msg},
	})
}

// NotFound sends a 404 JSON response.
func NotFound(c *gin.Context, code string, msg string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error: ErrorBody{Code: code, Message: msg},
	})
}

// Unauthorized sends a 401 JSON response.
func Unauthorized(c *gin.Context, code string, msg string) {
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error: ErrorBody{Code: code, Message: msg},
	})
}

// InternalError sends a 500 JSON response with an opaque message.
func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{Code: "INTERNAL_ERROR", Message: "internal error"},
	})
}
