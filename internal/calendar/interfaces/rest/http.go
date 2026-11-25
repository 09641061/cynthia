package rest

import (
	"time"

	"github.com/gin-gonic/gin"
	"go-service/internal/calendar/application"
	sharedrest "go-service/internal/shared/rest"
)

func parseFilter(c *gin.Context) (application.AppointmentFilter, bool) {
	var filter application.AppointmentFilter
	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			sharedrest.BadRequest(c, "The from query parameter must be RFC3339.")
			return filter, false
		}
		filter.From = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			sharedrest.BadRequest(c, "The to query parameter must be RFC3339.")
			return filter, false
		}
		filter.To = &t
	}
	if raw := c.Query("person_id"); raw != "" {
		if !validUUID(c, raw) {
			return filter, false
		}
		filter.PersonID = raw
	}
	if raw := c.Query("customer_id"); raw != "" {
		if !validUUID(c, raw) {
			return filter, false
		}
		filter.CustomerID = raw
	}
	filter.Status = c.Query("status")
	return filter, true
}

func validUUID(c *gin.Context, value string) bool {
	if _, ok := sharedrest.ParseUUID(value); !ok {
		sharedrest.BadRequest(c, "The identifier is invalid.")
		return false
	}
	return true
}

func validOptionalUUID(c *gin.Context, value *string) bool {
	if value == nil {
		return true
	}
	return validUUID(c, *value)
}
