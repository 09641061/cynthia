package domain

import (
	"strings"
	"time"
)

// DaySchedule represents opening status and HH:MM hours for a single weekday.
type DaySchedule struct {
	IsOpen    bool
	OpenTime  string
	CloseTime string
}

// SchedulePolicy defines the organization's timezone and weekly business hours.
type SchedulePolicy struct {
	Timezone string
	Days     map[time.Weekday]DaySchedule
}

// ValidateWithinBusinessHours ensures an interval falls on a single open business day within configured hours.
func ValidateWithinBusinessHours(startsAt, endsAt time.Time, policy SchedulePolicy) error {
	if strings.TrimSpace(policy.Timezone) == "" {
		return InvalidFieldError{Field: "starts_at", Reason: "organization timezone is not configured"}
	}
	loc, err := time.LoadLocation(policy.Timezone)
	if err != nil {
		return InvalidFieldError{Field: "starts_at", Reason: "invalid organization timezone"}
	}
	localStart := startsAt.In(loc)
	localEnd := endsAt.In(loc)
	sy, sm, sd := localStart.Date()
	ey, em, ed := localEnd.Date()
	if sy != ey || sm != em || sd != ed {
		return InvalidFieldError{Field: "starts_at", Reason: "appointment must start and end on the same business day"}
	}
	day, ok := policy.Days[localStart.Weekday()]
	if !ok || !day.IsOpen {
		return InvalidFieldError{Field: "starts_at", Reason: "organization is closed on this day"}
	}
	startHHMM := localStart.Format("15:04")
	endHHMM := localEnd.Format("15:04")
	if startHHMM < day.OpenTime || endHHMM > day.CloseTime ||
		(endHHMM == day.CloseTime && (localEnd.Second() > 0 || localEnd.Nanosecond() > 0)) {
		return InvalidFieldError{Field: "starts_at", Reason: "appointment must fall within organization business hours"}
	}
	return nil
}
