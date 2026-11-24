package crm

import (
	"context"
	"database/sql"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
	"go-service/internal/crm/infrastructure/persistence/postgres"
	"go-service/internal/crm/interfaces/rest"
)

// NewModule initializes the CRM module and registers its routes.
func NewModule(_ context.Context, db *sql.DB, router gin.IRouter) error {
	store := postgres.NewStore(db)
	customers := application.NewCustomerService(store)
	rest.RegisterRoutes(router.Group("/v1"), customers)
	return nil
}
