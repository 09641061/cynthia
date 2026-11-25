package organizations

import (
	"context"

	"go-service/internal/organizations/domain"
	"go-service/internal/organizations/infrastructure/persistence/postgres"
	sharedpostgres "go-service/internal/shared/postgres"
)

// ScheduleConfig exposes the organization's timezone and weekly business hours.
type ScheduleConfig struct {
	Timezone      string
	BusinessHours domain.BusinessHours
}

// ScheduleReader reads the singleton organization's schedule configuration.
type ScheduleReader struct {
	store *postgres.Store
	hasDB bool
}

// NewScheduleReader creates a reader for the organization's timezone and business hours.
func NewScheduleReader(db *sharedpostgres.Client) *ScheduleReader {
	if db == nil {
		return &ScheduleReader{}
	}
	return &ScheduleReader{store: postgres.NewStore(db), hasDB: true}
}

// GetScheduleConfig returns the organization's configured timezone and business hours.
func (r *ScheduleReader) GetScheduleConfig(ctx context.Context) (ScheduleConfig, error) {
	if !r.hasDB {
		return ScheduleConfig{}, nil
	}
	org, err := r.store.Get(ctx)
	if err != nil {
		return ScheduleConfig{}, err
	}
	return ScheduleConfig{
		Timezone:      org.Timezone,
		BusinessHours: org.BusinessHours,
	}, nil
}
