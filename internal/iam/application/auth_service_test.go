package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/iam/domain"
)

type fakeHasher struct{}

func (fakeHasher) Hash(pw string) (string, error)      { return "h:" + pw, nil }
func (fakeHasher) Verify(pw, enc string) (bool, error) { return enc == "h:"+pw, nil }

type fakeInviteStore struct {
	byHash map[string]PendingInvite
	byID   map[string]PendingInvite
}

func newFakeInviteStore() *fakeInviteStore {
	return &fakeInviteStore{byHash: map[string]PendingInvite{}, byID: map[string]PendingInvite{}}
}

func (f *fakeInviteStore) CreateInvite(_ context.Context, inv PendingInvite, _ time.Duration) error {
	f.byHash[inv.TokenHash] = inv
	f.byID[inv.ID] = inv
	return nil
}

func (f *fakeInviteStore) GetPending(_ context.Context, id string) (PendingInvite, error) {
	inv, ok := f.byID[id]
	if !ok {
		return PendingInvite{}, domain.ErrInvitationNotFound
	}
	return inv, nil
}

func (f *fakeInviteStore) VerifyInvite(_ context.Context, h string) (PendingInvite, error) {
	inv, ok := f.byHash[h]
	if !ok {
		return PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	return inv, nil
}

func (f *fakeInviteStore) ConsumeInvite(_ context.Context, h string) (PendingInvite, error) {
	inv, ok := f.byHash[h]
	if !ok {
		return PendingInvite{}, domain.ErrInviteExpiredOrUsed
	}
	delete(f.byHash, h)
	delete(f.byID, inv.ID)
	return inv, nil
}

func (f *fakeInviteStore) ListPending(_ context.Context) ([]PendingInvite, error) {
	out := make([]PendingInvite, 0, len(f.byID))
	for _, v := range f.byID {
		out = append(out, v)
	}
	return out, nil
}

func (f *fakeInviteStore) DeletePending(_ context.Context, id string) error {
	inv, ok := f.byID[id]
	if !ok {
		return domain.ErrInvitationNotFound
	}
	delete(f.byHash, inv.TokenHash)
	delete(f.byID, id)
	return nil
}

func (f *fakeInviteStore) ResendPending(_ context.Context, id, newHash string, ttl time.Duration) (PendingInvite, error) {
	inv, ok := f.byID[id]
	if !ok {
		return PendingInvite{}, domain.ErrInvitationNotFound
	}
	delete(f.byHash, inv.TokenHash)
	inv.TokenHash = newHash
	inv.ExpiresAt = time.Now().UTC().Add(ttl)
	f.byHash[newHash] = inv
	f.byID[id] = inv
	return inv, nil
}

type fakeMailer struct {
	lastToken string
	err       error
}

func (m *fakeMailer) SendInvitation(_ context.Context, _, _, rawToken string) error {
	m.lastToken = rawToken
	return m.err
}

func TestAuthServiceLoginRefreshAndOnboarding(t *testing.T) {
	users := newFakeUserStore()
	sessions := newFakeSessionStore()
	invites := newFakeInviteStore()
	mailer := &fakeMailer{}
	ownerID := uuid.NewString()
	users.users[ownerID] = domain.User{
		ID: ownerID, Email: "lucia@estudionube.mx", FirstName: "Lucia", LastName: "Mendez",
		PasswordHash: "h:TakoduDev2026!", Type: domain.UserTypeOwner,
	}

	authSvc := NewAuthService(users, sessions, invites, fakeHasher{}, 30*time.Minute, 24*time.Hour)
	pair, actor, err := authSvc.Login(t.Context(), "Lucia@EstudioNube.MX", "TakoduDev2026!")
	if err != nil || !actor.User.IsOwner() || pair.AccessToken == "" {
		t.Fatalf("Login = (%+v, %+v, %v)", pair, actor, err)
	}

	// Refresh rotates token; reusing old refresh token fails.
	pair2, err := authSvc.Refresh(t.Context(), pair.RefreshToken)
	if err != nil || pair2.RefreshToken == pair.RefreshToken {
		t.Fatalf("Refresh = (%+v, %v)", pair2, err)
	}
	if _, err := authSvc.Refresh(t.Context(), pair.RefreshToken); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("reused refresh = %v, want ErrInvalidToken", err)
	}

	// Invite -> Verify -> Accept (single-use) -> new Member.
	invSvc := NewInviteService(users, invites, mailer, 72*time.Hour)
	inv, err := invSvc.CreateInvitation(t.Context(), ownerID, "nuevo@estudionube.mx", "Diego", "Lopez")
	if err != nil || inv.Email != "nuevo@estudionube.mx" {
		t.Fatalf("CreateInvitation = (%+v, %v)", inv, err)
	}
	if _, err := authSvc.VerifyInvitation(t.Context(), mailer.lastToken); err != nil {
		t.Fatalf("VerifyInvitation: %v", err)
	}
	_, newActor, err := authSvc.AcceptInvitation(t.Context(), mailer.lastToken, "Diego", "Lopez", "SecurePass123!")
	if err != nil || newActor.User.IsOwner() {
		t.Fatalf("AcceptInvitation = (%+v, %v)", newActor, err)
	}
	if _, _, err := authSvc.AcceptInvitation(t.Context(), mailer.lastToken, "Diego", "Lopez", "SecurePass123!"); !errors.Is(err, domain.ErrInviteExpiredOrUsed) {
		t.Fatalf("second AcceptInvitation = %v, want ErrInviteExpiredOrUsed", err)
	}
}
