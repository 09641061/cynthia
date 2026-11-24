package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"go-service/internal/catalog"
	"go-service/internal/crm"
)

func newRouter(ctx context.Context, db *sql.DB) (*gin.Engine, error) {
	router := gin.Default()
	if err := catalog.NewModule(ctx, db, router); err != nil {
		return nil, fmt.Errorf("initialize catalog: %w", err)
	}
	if err := crm.NewModule(ctx, db, router); err != nil {
		return nil, fmt.Errorf("initialize crm: %w", err)
	}
	router.StaticFile("/openapi.yaml", "api/openapi.yaml")
	return router, nil
}
