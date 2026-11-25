package calendar

import (
	"fmt"
	"strings"
	"time"

	"go-service/internal/calendar/domain"
)

const availabilityStep = 30 * time.Minute

// AvailabilityQuery specifies parameters for single-day open slot calculation.
type AvailabilityQuery struct {
	Date                 string
	ServiceID            string
	PersonID             string
	ExcludeAppointmentID string
	CustomerID           string
}

// AvailabilitySlot represents a single bookable time slot for a staff member.
type AvailabilitySlot struct {
	StartAt    time.Time
	EndAt      time.Time
	PersonID   string
	PersonName string
}

func parseDateInPolicy(date string, policy domain.SchedulePolicy) (time.Time, *time.Location, error) {
	if strings.TrimSpace(policy.Timezone) == "" {
		return time.Time{}, nil, ErrOutsideBusinessHours
	}
	loc, err := time.LoadLocation(policy.Timezone)
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("load timezone: %w", err)
	}
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("invalid date %q: %w", date, err)
	}
	return day, loc, nil
}

func computeAvailability(
	now time.Time,
	policy domain.SchedulePolicy,
	date string,
	durationMinutes int,
	people []domain.Person,
	existing []domain.Appointment,
	query AvailabilityQuery,
) ([]AvailabilitySlot, error) {
	day, loc, err := parseDateInPolicy(date, policy)
	if err != nil {
		return nil, err
	}
	sched, ok := policy.Days[day.Weekday()]
	if !ok || !sched.IsOpen || len(people) == 0 || durationMinutes <= 0 {
		return []AvailabilitySlot{}, nil
	}
	filtered, err := filterActivePeople(people, query.PersonID)
	if err != nil {
		return nil, err
	}
	openAt, closeAt, err := dayBounds(day, loc, sched)
	if err != nil {
		return nil, err
	}
	dur := time.Duration(durationMinutes) * time.Minute
	nowLocal := now.In(loc)
	slots := make([]AvailabilitySlot, 0)
	for start := openAt; !start.Add(dur).After(closeAt); start = start.Add(availabilityStep) {
		if !start.After(nowLocal) {
			continue
		}
		end := start.Add(dur)
		for _, p := range filtered {
			if !hasBlockingOverlap(existing, p.ID, start, end, query) {
				slots = append(slots, AvailabilitySlot{
					StartAt:    start,
					EndAt:      end,
					PersonID:   p.ID,
					PersonName: p.Name,
				})
			}
		}
	}
	return slots, nil
}

func filterActivePeople(people []domain.Person, personID string) ([]domain.Person, error) {
	if personID == "" {
		return people, nil
	}
	for _, p := range people {
		if p.ID == personID {
			return []domain.Person{p}, nil
		}
	}
	return nil, ErrPersonNotFound
}

func dayBounds(day time.Time, loc *time.Location, sched domain.DaySchedule) (time.Time, time.Time, error) {
	o, err := time.Parse("15:04", sched.OpenTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	c, err := time.Parse("15:04", sched.CloseTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	y, m, d := day.Date()
	return time.Date(y, m, d, o.Hour(), o.Minute(), 0, 0, loc),
		time.Date(y, m, d, c.Hour(), c.Minute(), 0, 0, loc), nil
}

func hasBlockingOverlap(existing []domain.Appointment, personID string, start, end time.Time, q AvailabilityQuery) bool {
	for _, a := range existing {
		if a.PersonID != personID || !a.Status.IsBlocking() {
			continue
		}
		if q.ExcludeAppointmentID != "" && a.ID == q.ExcludeAppointmentID && q.CustomerID != "" && a.CustomerID == q.CustomerID {
			continue
		}
		if a.StartsAt.Before(end) && a.EndsAt.After(start) {
			return true
		}
	}
	return false
}
