package organizations

import (
	"context"

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

// WhatsAppConfigReader exposes the organization's WhatsApp configuration without leaking domain types.
type WhatsAppConfigReader struct {
	store *postgres.Store
	hasDB bool
}

// NewWhatsAppConfigReader creates a reader for the singleton organization's WhatsApp settings.
func NewWhatsAppConfigReader(db *sharedpostgres.Client) *WhatsAppConfigReader {
	if db == nil {
		return &WhatsAppConfigReader{}
	}
	return &WhatsAppConfigReader{store: postgres.NewStore(db), hasDB: true}
}

// GetWhatsAppConfig returns the configured Meta phone_number_id and enabled flag.
func (r *WhatsAppConfigReader) GetWhatsAppConfig(ctx context.Context) (string, bool, error) {
	if !r.hasDB {
		return "", false, nil
	}
	org, err := r.store.Get(ctx)
	if err != nil {
		return "", false, err
	}
	return org.WhatsAppPhoneNumberID, org.WhatsAppEnabled, nil
}
