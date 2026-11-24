package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func bindJSON(c *gin.Context, destination any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		badRequest(c)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		badRequest(c)
		return false
	}
	return true
}

func pathID(c *gin.Context, key string) (string, bool) {
	id, ok := normalizeID(c.Param(key))
	if !ok {
		badRequest(c)
		return "", false
	}
	return id, true
}

func ifMatch(c *gin.Context) (int64, bool) {
	value := c.GetHeader("If-Match")
	if value == "" {
		writeProblem(c, http.StatusPreconditionRequired, "Precondition Required", "If-Match is required.")
		return 0, false
	}
	if len(value) < 4 || !strings.HasPrefix(value, `"v`) || !strings.HasSuffix(value, `"`) {
		badRequest(c)
		return 0, false
	}
	version, err := strconv.ParseInt(value[2:len(value)-1], 10, 64)
	if err != nil || version < 1 || strconv.FormatInt(version, 10) != value[2:len(value)-1] {
		badRequest(c)
		return 0, false
	}
	return version, true
}

func entityTag(version int64) string { return `"v` + strconv.FormatInt(version, 10) + `"` }

func normalizeID(value string) (string, bool) {
	id, err := uuid.Parse(value)
	if err != nil {
		return "", false
	}
	return id.String(), true
}
