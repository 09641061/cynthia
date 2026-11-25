package iam

import (
	"context"
	"database/sql"
	"errors"

	sharedpostgres "go-service/internal/shared/postgres"
)

// ErrPersonNotFound indicates a referenced IAM user does not exist.
var ErrPersonNotFound = errors.New("person not found")

const schedulingAppearExists = `EXISTS (
	SELECT 1 FROM iam_member_roles mr
	JOIN iam_roles r ON r.id = mr.role_id
	WHERE mr.user_id = u.id AND 'scheduling.appear' = ANY(r.permissions)
)`

// PersonSummary is the minimal bookable-person projection exposed by IAM.
type PersonSummary struct {
	ID     string
	Name   string
	Active bool
}

// PersonReader reads bookable IAM users for cross-context calendar queries.
type PersonReader struct {
	db *sharedpostgres.Client
}

// NewPersonReader creates a reader for IAM bookable person records.
func NewPersonReader(db *sharedpostgres.Client) *PersonReader {
	return &PersonReader{db: db}
}

// ListActivePeople returns all IAM users assigned at least one role with scheduling.appear.
func (r *PersonReader) ListActivePeople(ctx context.Context) ([]PersonSummary, error) {
	if r.db == nil {
		return []PersonSummary{}, nil
	}
	var people []PersonSummary
	err := r.db.Do(ctx, func() error {
		rows, err := r.db.QueryContext(ctx, `
			SELECT u.id, btrim(u.first_name || ' ' || u.last_name) AS name, TRUE
			FROM iam_users u
			WHERE `+schedulingAppearExists+`
			ORDER BY u.first_name ASC, u.last_name ASC, u.id ASC`)
		if err != nil {
			return err
		}
		defer rows.Close()
		people = make([]PersonSummary, 0)
		for rows.Next() {
			var p PersonSummary
			if err := rows.Scan(&p.ID, &p.Name, &p.Active); err != nil {
				return err
			}
			people = append(people, p)
		}
		return rows.Err()
	})
	return people, err
}

// GetPerson fetches a single person summary by ID.
func (r *PersonReader) GetPerson(ctx context.Context, id string) (PersonSummary, error) {
	if r.db == nil {
		return PersonSummary{}, ErrPersonNotFound
	}
	var p PersonSummary
	err := r.db.Do(ctx, func() error {
		return r.db.QueryRowContext(ctx, `
			SELECT u.id, btrim(u.first_name || ' ' || u.last_name) AS name, `+schedulingAppearExists+`
			FROM iam_users u WHERE u.id = $1`, id,
		).Scan(&p.ID, &p.Name, &p.Active)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return PersonSummary{}, ErrPersonNotFound
	}
	return p, err
}

// GetPeopleByIDs batch-fetches person summaries keyed by user ID.
func (r *PersonReader) GetPeopleByIDs(ctx context.Context, ids []string) (map[string]PersonSummary, error) {
	result := make(map[string]PersonSummary)
	if r.db == nil || len(ids) == 0 {
		return result, nil
	}
	err := r.db.Do(ctx, func() error {
		rows, err := r.db.QueryContext(ctx, `
			SELECT u.id, btrim(u.first_name || ' ' || u.last_name) AS name, `+schedulingAppearExists+`
			FROM iam_users u WHERE u.id = ANY($1::uuid[])`, ids,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		result = make(map[string]PersonSummary)
		for rows.Next() {
			var p PersonSummary
			if err := rows.Scan(&p.ID, &p.Name, &p.Active); err != nil {
				return err
			}
			result[p.ID] = p
		}
		return rows.Err()
	})
	return result, err
}
