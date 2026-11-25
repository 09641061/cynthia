package application

import (
	"context"
	"slices"

	"go-service/internal/iam/domain"
)

type fakeUserStore struct {
	users map[string]domain.User
	roles map[string][]string
	all   map[string]domain.Role
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{
		users: make(map[string]domain.User),
		roles: make(map[string][]string),
		all:   make(map[string]domain.Role),
	}
}

func (f *fakeUserStore) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (f *fakeUserStore) GetActorByID(_ context.Context, userID string) (domain.Actor, error) {
	u, ok := f.users[userID]
	if !ok {
		return domain.Actor{}, domain.ErrUserNotFound
	}
	var roles []domain.Role
	for _, rid := range f.roles[userID] {
		if r, exists := f.all[rid]; exists {
			roles = append(roles, r)
		}
	}
	return domain.Actor{User: u, Roles: roles}, nil
}

func (f *fakeUserStore) ListActors(ctx context.Context) ([]domain.Actor, error) {
	out := make([]domain.Actor, 0, len(f.users))
	for id := range f.users {
		a, _ := f.GetActorByID(ctx, id)
		out = append(out, a)
	}
	return out, nil
}

func (f *fakeUserStore) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	for _, existing := range f.users {
		if existing.Email == u.Email {
			return domain.User{}, domain.ErrDuplicateEmail
		}
	}
	f.users[u.ID] = u
	return u, nil
}

func (f *fakeUserStore) UpdateProfile(_ context.Context, userID, fn, ln, pwHash string) (domain.User, error) {
	u, ok := f.users[userID]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	u.FirstName, u.LastName = fn, ln
	if pwHash != "" {
		u.PasswordHash = pwHash
	}
	f.users[userID] = u
	return u, nil
}

func (f *fakeUserStore) DeleteMember(_ context.Context, userID string) error {
	if _, ok := f.users[userID]; !ok {
		return domain.ErrUserNotFound
	}
	delete(f.users, userID)
	delete(f.roles, userID)
	return nil
}

func (f *fakeUserStore) ReplaceMemberRoles(ctx context.Context, userID string, roleIDs []string) (domain.Actor, error) {
	if _, ok := f.users[userID]; !ok {
		return domain.Actor{}, domain.ErrUserNotFound
	}
	f.roles[userID] = slices.Clone(roleIDs)
	return f.GetActorByID(ctx, userID)
}
