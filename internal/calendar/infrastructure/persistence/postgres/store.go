package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

const appointmentColumns = `id, person_id, customer_id, service_id, starts_at, ends_at, status, channel, notes, version, created_at, updated_at`

// Store implements calendar appointment persistence against PostgreSQL.
type Store struct{ db *sql.DB }

// NewStore creates a PostgreSQL calendar appointment store.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// ListAppointments returns appointments matching the given filter ordered by start time.
func (s *Store) ListAppointments(ctx context.Context, filter application.AppointmentFilter) ([]domain.Appointment, error) {
	where, args := buildAppointmentFilter(filter)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+appointmentColumns+` FROM calendar_appointments `+where+` ORDER BY starts_at ASC, id ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Appointment, 0)
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// GetAppointment fetches an appointment by ID.
func (s *Store) GetAppointment(ctx context.Context, id string) (domain.Appointment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+appointmentColumns+` FROM calendar_appointments WHERE id = $1`, id,
	)
	a, err := scanAppointment(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Appointment{}, domain.ErrNotFound
	}
	return a, err
}

// CreateAppointment inserts a new appointment, mapping exclusion violations to ErrTimeConflict.
func (s *Store) CreateAppointment(ctx context.Context, appt domain.Appointment) (domain.Appointment, error) {
	row := s.db.QueryRowContext(ctx,
		`INSERT INTO calendar_appointments
		 (id, person_id, customer_id, service_id, starts_at, ends_at, status, channel, notes, version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 1)
		 RETURNING `+appointmentColumns,
		appt.ID, appt.PersonID, appt.CustomerID, appt.ServiceID,
		appt.StartsAt.UTC(), appt.EndsAt.UTC(), string(appt.Status), string(appt.Channel), appt.Notes,
	)
	created, err := scanAppointment(row)
	if err != nil {
		return domain.Appointment{}, mapPGError(err)
	}
	return created, nil
}

// UpdateAppointment updates an appointment with optimistic locking and exclusion conflict detection.
func (s *Store) UpdateAppointment(ctx context.Context, expectedVersion int64, appt domain.Appointment) (domain.Appointment, error) {
	row := s.db.QueryRowContext(ctx,
		`UPDATE calendar_appointments
		 SET person_id = $1, customer_id = $2, service_id = $3,
		     starts_at = $4, ends_at = $5, status = $6, notes = $7,
		     version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $8 AND version = $9
		 RETURNING `+appointmentColumns,
		appt.PersonID, appt.CustomerID, appt.ServiceID,
		appt.StartsAt.UTC(), appt.EndsAt.UTC(), string(appt.Status), appt.Notes,
		appt.ID, expectedVersion,
	)
	updated, err := scanAppointment(row)
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.GetAppointment(ctx, appt.ID); getErr != nil {
			return domain.Appointment{}, getErr
		}
		return domain.Appointment{}, domain.ErrVersionConflict
	}
	if err != nil {
		return domain.Appointment{}, mapPGError(err)
	}
	return updated, nil
}

// DeleteAppointment removes an appointment by ID with optimistic locking.
func (s *Store) DeleteAppointment(ctx context.Context, id string, expectedVersion int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM calendar_appointments WHERE id = $1 AND version = $2`,
		id, expectedVersion,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM calendar_appointments WHERE id = $1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrVersionConflict
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAppointment(row rowScanner) (domain.Appointment, error) {
	var a domain.Appointment
	var status, channel string
	err := row.Scan(
		&a.ID, &a.PersonID, &a.CustomerID, &a.ServiceID,
		&a.StartsAt, &a.EndsAt, &status, &channel, &a.Notes,
		&a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return domain.Appointment{}, err
	}
	a.Status = domain.Status(status)
	a.Channel = domain.Channel(channel)
	return a, nil
}
