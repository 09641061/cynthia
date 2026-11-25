package organizations

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/organizations/infrastructure/persistence/postgres"
	"go-service/internal/organizations/interfaces/rest"
	sharedpostgres "go-service/internal/shared/postgres"
)

// NewModule initializes the organizations module and registers its routes.
func NewModule(db *sharedpostgres.Client, router gin.IRouter) {
	store := postgres.NewStore(db)
	rest.RegisterRoutes(router.Group("/v1"), store)
}
