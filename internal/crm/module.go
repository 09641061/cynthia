package crm

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"go-service/internal/crm/application"
	"go-service/internal/crm/infrastructure/persistence/postgres"
	"go-service/internal/crm/interfaces/rest"
	sharedpostgres "go-service/internal/shared/postgres"
)

// NewModule initializes the CRM module and registers its routes.
func NewModule(db *sharedpostgres.Client, router gin.IRouter) {
	var sqlDB *sql.DB
	if db != nil {
		sqlDB = db.DB
	}
	store := postgres.NewStore(sqlDB)
	customers := application.NewCustomerService(store)
	rest.RegisterRoutes(router.Group("/v1"), customers)
}
