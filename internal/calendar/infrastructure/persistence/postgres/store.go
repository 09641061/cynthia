package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
	sharedpostgres "go-service/internal/shared/postgres"
)

const appointmentColumns = `id, person_id, customer_id, service_id, starts_at, ends_at, status, channel, notes, created_at, updated_at`

// Store implements calendar appointment persistence against PostgreSQL.
type Store struct{ db *sharedpostgres.Client }

// NewStore creates a PostgreSQL calendar appointment store.
func NewStore(db *sharedpostgres.Client) *Store { return &Store{db: db} }

// ListAppointments returns appointments matching the given filter ordered by start time.
func (s *Store) ListAppointments(ctx context.Context, filter application.AppointmentFilter) ([]domain.Appointment, error) {
	where, args := buildAppointmentFilter(filter)
	var items []domain.Appointment
	err := s.db.Do(ctx, func() error {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+appointmentColumns+` FROM calendar_appointments `+where+` ORDER BY starts_at ASC, id ASC`,
			args...,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		items = make([]domain.Appointment, 0)
		for rows.Next() {
			a, err := scanAppointment(rows)
			if err != nil {
				return err
			}
			items = append(items, a)
		}
		return rows.Err()
	})
	return items, err
}

// GetAppointment fetches an appointment by ID.
func (s *Store) GetAppointment(ctx context.Context, id string) (domain.Appointment, error) {
	var a domain.Appointment
	err := s.db.Do(ctx, func() error {
		row := s.db.QueryRowContext(ctx, `SELECT `+appointmentColumns+` FROM calendar_appointments WHERE id = $1`, id)
		var err error
		a, err = scanAppointment(row)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Appointment{}, domain.ErrNotFound
	}
	return a, err
}

// CreateAppointment inserts a new appointment, mapping exclusion violations to ErrTimeConflict.
func (s *Store) CreateAppointment(ctx context.Context, appt domain.Appointment) (domain.Appointment, error) {
	var created domain.Appointment
	err := s.db.Do(ctx, func() error {
		row := s.db.QueryRowContext(ctx,
			`INSERT INTO calendar_appointments
			 (id, person_id, customer_id, service_id, starts_at, ends_at, status, channel, notes)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING `+appointmentColumns,
			appt.ID, appt.PersonID, appt.CustomerID, appt.ServiceID,
			appt.StartsAt.UTC(), appt.EndsAt.UTC(), string(appt.Status), string(appt.Channel), appt.Notes,
		)
		var err error
		created, err = scanAppointment(row)
		return err
	})
	if err != nil {
		return domain.Appointment{}, mapPGError(err)
	}
	return created, nil
}

// UpdateAppointment updates an appointment with exclusion conflict detection.
func (s *Store) UpdateAppointment(ctx context.Context, appt domain.Appointment) (domain.Appointment, error) {
	var updated domain.Appointment
	err := s.db.Do(ctx, func() error {
		row := s.db.QueryRowContext(ctx,
			`UPDATE calendar_appointments
			 SET person_id = $1, customer_id = $2, service_id = $3,
			     starts_at = $4, ends_at = $5, status = $6, notes = $7,
			     updated_at = CURRENT_TIMESTAMP
			 WHERE id = $8
			 RETURNING `+appointmentColumns,
			appt.PersonID, appt.CustomerID, appt.ServiceID,
			appt.StartsAt.UTC(), appt.EndsAt.UTC(), string(appt.Status), appt.Notes,
			appt.ID,
		)
		var err error
		updated, err = scanAppointment(row)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Appointment{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Appointment{}, mapPGError(err)
	}
	return updated, nil
}

// DeleteAppointment removes an appointment by ID.
func (s *Store) DeleteAppointment(ctx context.Context, id string) error {
	var affected int64
	err := s.db.Do(ctx, func() error {
		res, err := s.db.ExecContext(ctx, `DELETE FROM calendar_appointments WHERE id = $1`, id)
		if err != nil {
			return err
		}
		affected, err = res.RowsAffected()
		return err
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrNotFound
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
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return domain.Appointment{}, err
	}
	a.Status = domain.Status(status)
	a.Channel = domain.Channel(channel)
	return a, nil
}
