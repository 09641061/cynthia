package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sharedrest "go-service/internal/shared/rest"
)

func (h Handler) listCustomers(c *gin.Context) {
	page := sharedrest.QueryInt(c, "page", 1, 1, 100000)
	pageSize := sharedrest.QueryInt(c, "page_size", 20, 1, 100)
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
	if !sharedrest.BindJSON(c, &input) {
		return
	}
	created, err := h.customers.Create(c.Request.Context(), input.Name, input.Phone)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Location", "/v1/customers/"+created.ID)
	c.JSON(http.StatusCreated, toCustomerResource(created))
}

func (h Handler) getCustomer(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	customer, err := h.customers.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toCustomerResource(customer))
}

func (h Handler) updateCustomer(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
	if !ok {
		return
	}
	var input customerInput
	if !sharedrest.BindJSON(c, &input) {
		return
	}
	updated, err := h.customers.Update(c.Request.Context(), id, input.Name, input.Phone)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, toCustomerResource(updated))
}

func (h Handler) deleteCustomer(c *gin.Context) {
	id, ok := sharedrest.PathUUID(c, "id")
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
