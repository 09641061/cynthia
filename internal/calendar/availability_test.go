package calendar

import (
	"testing"
	"time"

	"go-service/internal/calendar/domain"
)

func testPolicy() domain.SchedulePolicy {
	return domain.SchedulePolicy{
		Timezone: "America/Bogota",
		Days: map[time.Weekday]domain.DaySchedule{
			time.Monday: {IsOpen: true, OpenTime: "09:00", CloseTime: "11:00"},
			time.Sunday: {IsOpen: false},
		},
	}
}

func TestComputeAvailabilitySlotsAndExclusionOwnership(t *testing.T) {
	loc, _ := time.LoadLocation("America/Bogota")
	now := time.Date(2026, 6, 1, 9, 15, 0, 0, loc) // Monday 09:15 -> 09:00 clipped
	people := []domain.Person{
		{ID: "p1", Name: "Lucia", Active: true},
		{ID: "p2", Name: "Elena", Active: true},
	}
	existing := []domain.Appointment{
		{
			ID: "a1", PersonID: "p1", CustomerID: "cust-1", Status: domain.StatusConfirmed,
			StartsAt: time.Date(2026, 6, 1, 9, 30, 0, 0, loc),
			EndsAt:   time.Date(2026, 6, 1, 10, 0, 0, 0, loc),
		},
	}
	slots, err := computeAvailability(now, testPolicy(), "2026-06-01", 30, people, existing, AvailabilityQuery{
		Date: "2026-06-01", PersonID: "p1", ExcludeAppointmentID: "a1", CustomerID: "cust-other",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slots) != 2 || slots[0].StartAt.Hour() != 10 {
		t.Fatalf("expected 09:30 blocked for non-owner, got %+v", slots)
	}
	slotsOwned, err := computeAvailability(now, testPolicy(), "2026-06-01", 30, people, existing, AvailabilityQuery{
		Date: "2026-06-01", PersonID: "p1", ExcludeAppointmentID: "a1", CustomerID: "cust-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slotsOwned) != 3 || slotsOwned[0].StartAt.Minute() != 30 {
		t.Fatalf("expected 09:30 excluded for owner, got %+v", slotsOwned)
	}
}

func TestComputeAvailabilityClosedOrNoPeople(t *testing.T) {
	loc, _ := time.LoadLocation("America/Bogota")
	now := time.Date(2026, 5, 31, 8, 0, 0, 0, loc) // Sunday closed
	slots, err := computeAvailability(now, testPolicy(), "2026-05-31", 30, []domain.Person{{ID: "p1", Active: true}}, nil, AvailabilityQuery{Date: "2026-05-31"})
	if err != nil || len(slots) != 0 {
		t.Fatalf("expected empty on closed day, got %v, %v", slots, err)
	}
	slots, err = computeAvailability(now, testPolicy(), "2026-06-01", 30, nil, nil, AvailabilityQuery{Date: "2026-06-01"})
	if err != nil || len(slots) != 0 {
		t.Fatalf("expected empty with 0 active staff, got %v, %v", slots, err)
	}
}
