package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h Handler) listCustomers(c *gin.Context) {
	page, ok := queryInt(c, "page", 1)
	if !ok {
		return
	}
	pageSize, ok := queryInt(c, "page_size", 20)
	if !ok {
		return
	}
	result, err := h.customers.List(c.Request.Context(), c.Query("q"), page, pageSize)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toCustomerPageResource(result))
}

func (h Handler) createCustomer(c *gin.Context) {
	var input customerInput
	if !bindJSON(c, &input) {
		return
	}
	created, err := h.customers.Create(c.Request.Context(), input.Name, input.Phone)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/v1/customers/"+created.ID)
	c.Header("ETag", entityTag(created.Version))
	c.JSON(http.StatusCreated, toCustomerResource(created))
}

func (h Handler) getCustomer(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	customer, err := h.customers.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(customer.Version))
	c.JSON(http.StatusOK, toCustomerResource(customer))
}

func (h Handler) updateCustomer(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	version, ok := ifMatch(c)
	if !ok {
		return
	}
	var input customerInput
	if !bindJSON(c, &input) {
		return
	}
	updated, err := h.customers.Update(c.Request.Context(), id, version, input.Name, input.Phone)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("ETag", entityTag(updated.Version))
	c.JSON(http.StatusOK, toCustomerResource(updated))
}

func (h Handler) deleteCustomer(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	if err := h.customers.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusNoContent)
}
