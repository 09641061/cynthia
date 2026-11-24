package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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

// IfMatch parses the strong ETag version from the If-Match request header.
func IfMatch(c *gin.Context) (int64, bool) {
	value := c.GetHeader("If-Match")
	if value == "" {
		WriteProblem(c, http.StatusPreconditionRequired, "Precondition Required", "If-Match is required.")
		return 0, false
	}
	if len(value) < 4 || !strings.HasPrefix(value, `"v`) || !strings.HasSuffix(value, `"`) {
		BadRequest(c, "The If-Match header is invalid.")
		return 0, false
	}
	version, err := strconv.ParseInt(value[2:len(value)-1], 10, 64)
	if err != nil || version < 1 || strconv.FormatInt(version, 10) != value[2:len(value)-1] {
		BadRequest(c, "The If-Match header is invalid.")
		return 0, false
	}
	return version, true
}

// EntityTag formats a resource version as a strong ETag header value.
func EntityTag(version int64) string {
	return `"v` + strconv.FormatInt(version, 10) + `"`
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
