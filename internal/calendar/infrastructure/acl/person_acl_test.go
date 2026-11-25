package acl

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/calendar/domain"
	"go-service/internal/iam"
)

func TestPersonAdapterNilReader(t *testing.T) {
	adapter := NewPersonAdapter(iam.NewPersonReader(nil))
	people, err := adapter.ListActivePeople(t.Context())
	if err != nil || len(people) != 0 {
		t.Fatalf("ListActivePeople(nil) = (%v, %v), want ([], nil)", people, err)
	}
	if _, err := adapter.GetPerson(t.Context(), uuid.NewString()); !errors.Is(err, domain.ErrPersonNotFound) {
		t.Fatalf("GetPerson(nil) = %v, want ErrPersonNotFound", err)
	}
	byID, err := adapter.GetPeopleByIDs(t.Context(), []string{uuid.NewString()})
	if err != nil || len(byID) != 0 {
		t.Fatalf("GetPeopleByIDs(nil) = (%v, %v), want ({}, nil)", byID, err)
	}
}
