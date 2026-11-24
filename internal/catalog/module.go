package catalog

import (
	"context"
	"database/sql"

	"github.com/gin-gonic/gin"
	"go-service/internal/catalog/application"
	"go-service/internal/catalog/infrastructure/persistence/postgres"
	"go-service/internal/catalog/interfaces/rest"
)

// NewModule initializes the catalog module and registers its routes.
func NewModule(ctx context.Context, db *sql.DB, router gin.IRouter) error {
	if err := postgres.Migrate(ctx, db); err != nil {
		return err
	}
	store := postgres.NewStore(db)
	useCases := application.NewCatalog(store)
	rest.RegisterRoutes(router.Group("/v1"), useCases)
	return nil
}
