package catalog

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/catalog/application"
	"go-service/internal/catalog/infrastructure/persistence/postgres"
	"go-service/internal/catalog/interfaces/rest"
	sharedpostgres "go-service/internal/shared/postgres"
)

// NewModule initializes the catalog module and registers its routes.
func NewModule(db *sharedpostgres.Client, router gin.IRouter) {
	store := postgres.NewStore(db)
	useCases := application.NewCatalog(store)
	rest.RegisterRoutes(router.Group("/v1"), useCases)
}
