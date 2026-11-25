package bootstrap

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
	"go-service/internal/shared/postgres"
)

func newRouter(db *postgres.Client) *gin.Engine {
	router := gin.Default()
	catalog.NewModule(db, router)
	crm.NewModule(db, router)
	organizations.NewModule(db, router)
	router.StaticFile("/openapi.yaml", "api/openapi.yaml")
	return router
}
