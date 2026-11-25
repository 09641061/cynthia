package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/iam/domain"
)

type fakeRoleStore struct {
	roles map[string]domain.Role
	order []string
}

func newFakeRoleStore() *fakeRoleStore {
	return &fakeRoleStore{roles: map[string]domain.Role{}}
}

func (f *fakeRoleStore) ListRoles(_ context.Context) ([]domain.Role, error) {
	out := make([]domain.Role, 0, len(f.order))
	for _, id := range f.order {
		if r, ok := f.roles[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeRoleStore) GetRole(_ context.Context, id string) (domain.Role, error) {
	r, ok := f.roles[id]
	if !ok {
		return domain.Role{}, domain.ErrRoleNotFound
	}
	return r, nil
}

func (f *fakeRoleStore) CountRoles(_ context.Context) (int, error) { return len(f.roles), nil }

func (f *fakeRoleStore) CreateRole(_ context.Context, r domain.Role) (domain.Role, error) {
	r.Position = len(f.order) + 1
	f.roles[r.ID] = r
	f.order = append(f.order, r.ID)
	return r, nil
}

func (f *fakeRoleStore) UpdateRole(_ context.Context, id, name string, perms []string) (domain.Role, error) {
	r, ok := f.roles[id]
	if !ok {
		return domain.Role{}, domain.ErrRoleNotFound
	}
	r.Name, r.Permissions = name, perms
	f.roles[id] = r
	return r, nil
}

func (f *fakeRoleStore) DeleteRole(_ context.Context, id string) error {
	if _, ok := f.roles[id]; !ok {
		return domain.ErrRoleNotFound
	}
	delete(f.roles, id)
	return nil
}

func (f *fakeRoleStore) ReorderRoles(ctx context.Context, orderedIDs []string) ([]domain.Role, error) {
	f.order = orderedIDs
	for i, id := range orderedIDs {
		r := f.roles[id]
		r.Position = i + 1
		f.roles[id] = r
	}
	return f.ListRoles(ctx)
}

func TestRoleCapAndInviteRollbackOnMailerFailure(t *testing.T) {
	users := newFakeUserStore()
	roles := newFakeRoleStore()
	ownerID := uuid.NewString()
	users.users[ownerID] = domain.User{ID: ownerID, Type: domain.UserTypeOwner}

	roleSvc := NewRoleService(users, roles)
	for i := range domain.MaxActiveRoles {
		if _, err := roleSvc.CreateRole(t.Context(), ownerID, "Role "+uuid.NewString()[:8], []string{domain.PermCatalogManage}); err != nil {
			t.Fatalf("create role %d: %v", i, err)
		}
	}
	if _, err := roleSvc.CreateRole(t.Context(), ownerID, "Overflow", nil); !errors.Is(err, domain.ErrRoleLimitExceeded) {
		t.Fatalf("51st role = %v, want ErrRoleLimitExceeded", err)
	}

	invites := newFakeInviteStore()
	mailer := &fakeMailer{err: errors.New("resend down")}
	invSvc := NewInviteService(users, invites, mailer, time.Hour)
	if _, err := invSvc.CreateInvitation(t.Context(), ownerID, "x@example.com", "Ana", "Perez"); err == nil {
		t.Fatal("expected error when mailer fails")
	}
	if pending, _ := invites.ListPending(t.Context()); len(pending) != 0 {
		t.Fatalf("expected invite rollback on mailer failure, got %d pending", len(pending))
	}
}

func TestCreateAndResendInvitationPreservesTokenOnMailerFailure(t *testing.T) {
	users := newFakeUserStore()
	ownerID := uuid.NewString()
	users.users[ownerID] = domain.User{ID: ownerID, Type: domain.UserTypeOwner}
	invites := newFakeInviteStore()
	mailer := &fakeMailer{}
	invSvc := NewInviteService(users, invites, mailer, time.Hour)

	inv, err := invSvc.CreateInvitation(t.Context(), ownerID, "new@example.com", "  ", "")
	if err != nil {
		t.Fatalf("CreateInvitation with empty names failed: %v", err)
	}
	if inv.FirstName != "" || inv.LastName != "" {
		t.Fatalf("expected trimmed empty names, got (%q, %q)", inv.FirstName, inv.LastName)
	}
	if _, err := invSvc.CreateInvitation(t.Context(), ownerID, "NEW@example.com", "Ana", "Gomez"); !errors.Is(err, domain.ErrDuplicateInvitation) {
		t.Fatalf("duplicate pending invite = %v, want ErrDuplicateInvitation", err)
	}

	origHash := inv.TokenHash
	mailer.err = errors.New("resend down")
	if _, err := invSvc.ResendInvitation(t.Context(), ownerID, inv.ID); err == nil {
		t.Fatal("expected ResendInvitation to fail when mailer fails")
	}
	stored, err := invites.GetPending(t.Context(), inv.ID)
	if err != nil || stored.TokenHash != origHash {
		t.Fatalf("expected original token hash preserved after failed resend, got (%+v, %v)", stored, err)
	}
}
