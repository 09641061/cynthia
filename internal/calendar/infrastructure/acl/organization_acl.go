package acl

import (
	"context"
	"time"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
	"go-service/internal/organizations"
)

var _ application.SchedulePolicyReader = OrganizationAdapter{}

// OrganizationAdapter translates organization schedule config into calendar's SchedulePolicyReader port.
type OrganizationAdapter struct {
	reader *organizations.ScheduleReader
}

// NewOrganizationAdapter wraps an organizations ScheduleReader for calendar use cases.
func NewOrganizationAdapter(reader *organizations.ScheduleReader) OrganizationAdapter {
	return OrganizationAdapter{reader: reader}
}

// GetSchedulePolicy fetches and translates the organization's timezone and business hours.
func (a OrganizationAdapter) GetSchedulePolicy(ctx context.Context) (domain.SchedulePolicy, error) {
	if a.reader == nil {
		return domain.SchedulePolicy{}, nil
	}
	cfg, err := a.reader.GetScheduleConfig(ctx)
	if err != nil {
		return domain.SchedulePolicy{}, err
	}
	if cfg.Timezone == "" {
		return domain.SchedulePolicy{}, nil
	}
	bh := cfg.BusinessHours
	return domain.SchedulePolicy{
		Timezone: cfg.Timezone,
		Days: map[time.Weekday]domain.DaySchedule{
			time.Monday:    {IsOpen: bh.Monday.IsOpen, OpenTime: bh.Monday.OpenTime, CloseTime: bh.Monday.CloseTime},
			time.Tuesday:   {IsOpen: bh.Tuesday.IsOpen, OpenTime: bh.Tuesday.OpenTime, CloseTime: bh.Tuesday.CloseTime},
			time.Wednesday: {IsOpen: bh.Wednesday.IsOpen, OpenTime: bh.Wednesday.OpenTime, CloseTime: bh.Wednesday.CloseTime},
			time.Thursday:  {IsOpen: bh.Thursday.IsOpen, OpenTime: bh.Thursday.OpenTime, CloseTime: bh.Thursday.CloseTime},
			time.Friday:    {IsOpen: bh.Friday.IsOpen, OpenTime: bh.Friday.OpenTime, CloseTime: bh.Friday.CloseTime},
			time.Saturday:  {IsOpen: bh.Saturday.IsOpen, OpenTime: bh.Saturday.OpenTime, CloseTime: bh.Saturday.CloseTime},
			time.Sunday:    {IsOpen: bh.Sunday.IsOpen, OpenTime: bh.Sunday.OpenTime, CloseTime: bh.Sunday.CloseTime},
		},
	}, nil
}
