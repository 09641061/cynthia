package domain_test

import (
	"errors"
	"testing"
	"time"

	"go-service/internal/calendar/domain"
)

func testPolicy(tz string) domain.SchedulePolicy {
	weekday := domain.DaySchedule{IsOpen: true, OpenTime: "09:00", CloseTime: "18:00"}
	closed := domain.DaySchedule{IsOpen: false, OpenTime: "09:00", CloseTime: "18:00"}
	return domain.SchedulePolicy{
		Timezone: tz,
		Days: map[time.Weekday]domain.DaySchedule{
			time.Monday:    weekday,
			time.Tuesday:   weekday,
			time.Wednesday: weekday,
			time.Thursday:  weekday,
			time.Friday:    weekday,
			time.Saturday:  closed,
			time.Sunday:    closed,
		},
	}
}

func TestValidateWithinBusinessHours(t *testing.T) {
	utcPolicy := testPolicy("UTC")
	mxPolicy := testPolicy("America/Mexico_City")

	cases := []struct {
		name    string
		policy  domain.SchedulePolicy
		start   time.Time
		end     time.Time
		wantErr bool
	}{
		{
			name:   "open hours pass on exact boundaries",
			policy: utcPolicy,
			start:  time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC),
			end:    time.Date(2026, 11, 25, 18, 0, 0, 0, time.UTC),
		},
		{
			name:    "before open fails",
			policy:  utcPolicy,
			start:   time.Date(2026, 11, 25, 8, 30, 0, 0, time.UTC),
			end:     time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC),
			wantErr: true,
		},
		{
			name:    "after close fails",
			policy:  utcPolicy,
			start:   time.Date(2026, 11, 25, 17, 30, 0, 0, time.UTC),
			end:     time.Date(2026, 11, 25, 18, 0, 1, 0, time.UTC),
			wantErr: true,
		},
		{
			name:    "closed day fails",
			policy:  utcPolicy,
			start:   time.Date(2026, 11, 28, 10, 0, 0, 0, time.UTC),
			end:     time.Date(2026, 11, 28, 10, 30, 0, 0, time.UTC),
			wantErr: true,
		},
		{
			name:    "cross day interval fails",
			policy:  utcPolicy,
			start:   time.Date(2026, 11, 25, 17, 30, 0, 0, time.UTC),
			end:     time.Date(2026, 11, 26, 9, 30, 0, 0, time.UTC),
			wantErr: true,
		},
		{
			name:   "mexico city timezone conversion passes",
			policy: mxPolicy,
			start:  time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC),
			end:    time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "mexico city before local open fails",
			policy:  mxPolicy,
			start:   time.Date(2026, 11, 25, 14, 30, 0, 0, time.UTC),
			end:     time.Date(2026, 11, 25, 15, 30, 0, 0, time.UTC),
			wantErr: true,
		},
		{
			name:    "invalid timezone fails",
			policy:  testPolicy("Invalid/Zone"),
			start:   time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC),
			end:     time.Date(2026, 11, 25, 10, 30, 0, 0, time.UTC),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateWithinBusinessHours(tc.start, tc.end, tc.policy)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			var fieldErr domain.InvalidFieldError
			if !errors.As(err, &fieldErr) || fieldErr.Field != "starts_at" {
				t.Fatalf("got err %v, want InvalidFieldError(starts_at)", err)
			}
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("expected ErrInvalid unwrap, got %v", err)
			}
		})
	}
}
