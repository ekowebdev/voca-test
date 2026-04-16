package util

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationLinks contains links for navigation
type PaginationLinks struct {
	Current string `json:"current"`
	First   string `json:"first"`
	Last    string `json:"last"`
	Prev    string `json:"prev,omitempty"`
	Next    string `json:"next,omitempty"`
}

// PaginationMeta contains pagination information
type PaginationMeta struct {
	CurrentPage int             `json:"current_page"`
	PerPage     int             `json:"per_page"`
	TotalItems  int64           `json:"total_items"`
	TotalPages  int             `json:"total_pages"`
	Links       PaginationLinks `json:"links"`
}

// GenerateLinks populates the Links field based on the request context
func (m *PaginationMeta) GenerateLinks(c *gin.Context) {
	// Build the base URL
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := c.Request.Host
	if host == "" {
		host = "localhost" // Fallback for tests or misconfigured servers
	}
	path := c.Request.URL.Path

	baseURL := fmt.Sprintf("%s://%s%s", scheme, host, path)
	u, _ := url.Parse(baseURL)
	q := c.Request.URL.Query()

	buildURL := func(page int) string {
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()
		return u.String()
	}

	m.Links.Current = buildURL(m.CurrentPage)
	m.Links.First = buildURL(1)
	m.Links.Last = buildURL(m.TotalPages)

	if m.CurrentPage > 1 {
		m.Links.Prev = buildURL(m.CurrentPage - 1)
	}
	if m.CurrentPage < m.TotalPages {
		m.Links.Next = buildURL(m.CurrentPage + 1)
	}
}

// APIResponse is the standard structure for all API responses
type APIResponse struct {
	Status  string          `json:"status"`            // "success" or "error"
	Message string          `json:"message,omitempty"` // General message
	Data    interface{}     `json:"data,omitempty"`    // Actual payload for success
	Meta    *PaginationMeta `json:"meta,omitempty"`    // Pagination metadata
	Errors  interface{}     `json:"errors,omitempty"`  // Specific error details (e.g., validation)
}

// SuccessResponse sends a standard success response
func SuccessResponse(c *gin.Context, statusCode int, data interface{}, message string) {
	c.JSON(statusCode, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// SuccessResponseWithPagination sends a success response with pagination metadata
func SuccessResponseWithPagination(c *gin.Context, statusCode int, data interface{}, meta PaginationMeta, message string) {
	c.JSON(statusCode, APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    &meta,
	})
}

// ErrorResponse sends a standard error response
func ErrorResponse(c *gin.Context, statusCode int, message string, errors interface{}) {
	c.JSON(statusCode, APIResponse{
		Status:  "error",
		Message: message,
		Errors:  errors,
	})
}
