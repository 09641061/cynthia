package postgres

import (
	"context"
	"database/sql"
	"errors"

	"go-service/internal/calendar/domain"
)

// PersonStore resolves bookable people from calendar_people in PostgreSQL.
type PersonStore struct{ db *sql.DB }

// NewPersonStore creates a PostgreSQL person resolver.
func NewPersonStore(db *sql.DB) *PersonStore { return &PersonStore{db: db} }

// ListActivePeople returns all active bookable people ordered by name.
func (s *PersonStore) ListActivePeople(ctx context.Context) ([]domain.Person, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, active FROM calendar_people WHERE active = true ORDER BY name ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	people := make([]domain.Person, 0)
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Active); err != nil {
			return nil, err
		}
		people = append(people, p)
	}
	return people, rows.Err()
}

// GetPerson fetches a person by ID.
func (s *PersonStore) GetPerson(ctx context.Context, id string) (domain.Person, error) {
	var p domain.Person
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, active FROM calendar_people WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Person{}, domain.ErrPersonNotFound
	}
	return p, err
}

// GetPeopleByIDs batch-fetches people keyed by ID.
func (s *PersonStore) GetPeopleByIDs(ctx context.Context, ids []string) (map[string]domain.Person, error) {
	result := make(map[string]domain.Person)
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, active FROM calendar_people WHERE id = ANY($1::uuid[])`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Active); err != nil {
			return nil, err
		}
		result[p.ID] = p
	}
	return result, rows.Err()
}
