package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Problem represents an RFC 9457 Problem Details response body.
type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Detail   string         `json:"detail"`
	Instance string         `json:"instance"`
	Errors   []ProblemField `json:"errors"`
}

// ProblemField identifies a single invalid input field in a Problem response.
type ProblemField struct {
	Field   string `json:"field"`
	Pointer string `json:"pointer"`
	Code    string `json:"code"`
	Reason  string `json:"reason"`
}

// BindJSON decodes a single JSON object from the request body, rejecting unknown fields.
func BindJSON(c *gin.Context, destination any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		BadRequest(c, "The request body or header is invalid.")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		BadRequest(c, "The request body or header is invalid.")
		return false
	}
	return true
}

// ParseUUID validates a UUID string and returns its canonical form.
func ParseUUID(raw string) (string, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return id.String(), true
}

// PathUUID parses a UUID path parameter, writing a 400 Problem response on failure.
func PathUUID(c *gin.Context, key string) (string, bool) {
	id, ok := ParseUUID(c.Param(key))
	if !ok {
		BadRequest(c, "The "+key+" path parameter must be a valid UUID.")
		return "", false
	}
	return id, true
}

// ParseUUIDSlice validates each element of ids as a UUID, writing a 400 Problem response on failure.
func ParseUUIDSlice(c *gin.Context, ids []string) bool {
	for _, raw := range ids {
		if _, ok := ParseUUID(raw); !ok {
			BadRequest(c, "All identifiers must be valid UUIDs.")
			return false
		}
	}
	return true
}

// QueryInt parses an integer query parameter clamped to [min, max], returning fallback if absent or invalid.
func QueryInt(c *gin.Context, key string, fallback, min, max int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// BadRequest aborts the request with a 400 Problem Details response.
func BadRequest(c *gin.Context, detail string) {
	WriteProblem(c, http.StatusBadRequest, "Bad Request", detail)
}

// WriteProblem aborts the request with an RFC 9457 Problem Details response.
func WriteProblem(c *gin.Context, status int, title, detail string) {
	WriteProblemFields(c, status, title, detail, []ProblemField{})
}

// WriteProblemFields aborts the request with an RFC 9457 Problem Details response including field errors.
func WriteProblemFields(c *gin.Context, status int, title, detail string, fields []ProblemField) {
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, Problem{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: c.Request.URL.Path,
		Errors:   fields,
	})
}
