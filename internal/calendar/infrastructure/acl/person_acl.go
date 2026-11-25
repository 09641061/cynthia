package acl

import (
	"context"
	"errors"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
	"go-service/internal/iam"
)

var _ application.PersonResolver = PersonAdapter{}

// PersonAdapter translates IAM bookable users into calendar's PersonResolver port.
type PersonAdapter struct {
	reader *iam.PersonReader
}

// NewPersonAdapter wraps an IAM PersonReader for calendar use cases.
func NewPersonAdapter(reader *iam.PersonReader) PersonAdapter {
	return PersonAdapter{reader: reader}
}

// ListActivePeople returns all IAM users with an active scheduling.appear grant.
func (a PersonAdapter) ListActivePeople(ctx context.Context) ([]domain.Person, error) {
	if a.reader == nil {
		return []domain.Person{}, nil
	}
	raw, err := a.reader.ListActivePeople(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Person, 0, len(raw))
	for _, p := range raw {
		out = append(out, domain.Person{ID: p.ID, Name: p.Name, Active: p.Active})
	}
	return out, nil
}

// GetPerson fetches and translates a single person summary.
func (a PersonAdapter) GetPerson(ctx context.Context, id string) (domain.Person, error) {
	if a.reader == nil {
		return domain.Person{}, domain.ErrPersonNotFound
	}
	p, err := a.reader.GetPerson(ctx, id)
	if errors.Is(err, iam.ErrPersonNotFound) {
		return domain.Person{}, domain.ErrPersonNotFound
	}
	if err != nil {
		return domain.Person{}, err
	}
	return domain.Person{ID: p.ID, Name: p.Name, Active: p.Active}, nil
}

// GetPeopleByIDs batch-fetches and translates person summaries keyed by ID.
func (a PersonAdapter) GetPeopleByIDs(ctx context.Context, ids []string) (map[string]domain.Person, error) {
	if a.reader == nil {
		return map[string]domain.Person{}, nil
	}
	raw, err := a.reader.GetPeopleByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Person, len(raw))
	for k, v := range raw {
		out[k] = domain.Person{ID: v.ID, Name: v.Name, Active: v.Active}
	}
	return out, nil
}
