package httpapi

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"plaiflow/api/internal/auth"
	"plaiflow/api/internal/tenant"
)

type sessionStore struct{ session auth.Session }

func (*sessionStore) CreateAuthTransaction(context.Context, auth.Transaction) error { return nil }
func (*sessionStore) ConsumeAuthTransaction(context.Context, []byte, time.Time) (auth.Transaction, error) {
	return auth.Transaction{}, auth.ErrAuthTransaction
}
func (*sessionStore) Login(context.Context, auth.ExternalIdentity, string, string, time.Time) (auth.LoginResult, error) {
	return auth.LoginResult{}, nil
}
func (*sessionStore) CreateSession(context.Context, auth.SessionRecord) error { return nil }
func (s *sessionStore) ResolveSession(context.Context, []byte, time.Time) (auth.Session, error) {
	return s.session, nil
}
func (*sessionStore) RevokeSession(context.Context, string, time.Time) error            { return nil }
func (*sessionStore) RevokeUserSessions(context.Context, string, time.Time) error       { return nil }
func (*sessionStore) LinkIdentityAndRotate(context.Context, auth.LinkRequest) error     { return nil }
func (*sessionStore) ReauthenticateAndRotate(context.Context, auth.ReauthRequest) error { return nil }
func (*sessionStore) UnlinkIdentityAndRotate(context.Context, auth.UnlinkRequest) error { return nil }
func (*sessionStore) ListIdentities(context.Context, string) ([]auth.IdentityState, error) {
	return nil, nil
}

type tenantStore struct {
	created tenant.Organization
	allowed string
}

func (s *tenantStore) CreateOrganization(_ context.Context, _, id, name string, _ time.Time) (tenant.Organization, error) {
	s.created = tenant.Organization{ID: id, Name: name, Role: tenant.Owner}
	return s.created, nil
}
func (*tenantStore) ListOrganizations(context.Context, string) ([]tenant.Organization, error) {
	return nil, nil
}
func (s *tenantStore) ResolveMembership(_ context.Context, userID, organizationID string) (tenant.Membership, error) {
	if organizationID != s.allowed {
		return tenant.Membership{}, tenant.ErrNotFound
	}
	return tenant.Membership{OrganizationID: organizationID, UserID: userID, Role: tenant.Owner}, nil
}
func (*tenantStore) ListMemberships(context.Context, string, string) ([]tenant.Membership, error) {
	return nil, nil
}
func (*tenantStore) CreateInvitation(context.Context, tenant.InviteCreate) error { return nil }
func (*tenantStore) ClaimInvitation(context.Context, tenant.InviteClaim) (tenant.Invitation, error) {
	return tenant.Invitation{}, nil
}
func (*tenantStore) AcceptInvitation(context.Context, []byte, string, time.Time) (tenant.Organization, error) {
	return tenant.Organization{}, nil
}
func (*tenantStore) RevokeInvitation(context.Context, string, string, string, time.Time) error {
	return nil
}
func (*tenantStore) ChangeRole(context.Context, string, string, string, tenant.Role, time.Time) error {
	return nil
}
func (*tenantStore) TransferOwnership(context.Context, string, string, string, time.Time) error {
	return nil
}
func (*tenantStore) RemoveMembership(context.Context, string, string, string, time.Time) error {
	return nil
}
func (*tenantStore) LeaveOrganization(context.Context, string, string, time.Time) error { return nil }
func (*tenantStore) CreateLineLinkCode(context.Context, tenant.LineCodeCreate) error    { return nil }
func (*tenantStore) ListLineConnections(context.Context, string, string) ([]tenant.LineConnection, error) {
	return nil, nil
}
func (*tenantStore) DisconnectLineConnection(context.Context, string, string, string, time.Time) error {
	return nil
}

func TestOrganizationCreationAndCrossTenantRejection(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	authService, err := auth.NewService(auth.ServiceConfig{WebOrigin: "https://app.example", Now: func() time.Time { return now }}, &sessionStore{session: auth.Session{
		ID: "session-1", UserID: "user-1", CSRFHash: hash("csrf"), AuthenticatedAt: now, IdleExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(time.Hour),
	}})
	if err != nil {
		t.Fatal(err)
	}
	tenants := &tenantStore{allowed: "11111111-1111-4111-8111-111111111111"}
	handler := New(Config{LineSecret: "secret", LineChannel: "channel", DashboardTokens: []string{"dashboard"}, Auth: authService, Tenants: tenants}, &fakeStore{})

	form := url.Values{"name": {"บริษัท ปลายโฟลว์"}, "csrf_token": {"csrf"}}
	create := httptest.NewRequest(http.MethodPost, "https://app.example/v1/organizations", strings.NewReader(form.Encode()))
	create.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	create.Header.Set("Origin", "https://app.example")
	create.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session"})
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, create)
	if createResponse.Code != http.StatusCreated || tenants.created.Name != "บริษัท ปลายโฟลว์" || tenants.created.Role != tenant.Owner {
		t.Fatalf("create status=%d organization=%+v", createResponse.Code, tenants.created)
	}

	spoof := httptest.NewRequest(http.MethodGet, "https://app.example/v1/o/22222222-2222-4222-8222-222222222222", nil)
	spoof.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session"})
	spoofResponse := httptest.NewRecorder()
	handler.ServeHTTP(spoofResponse, spoof)
	if spoofResponse.Code != http.StatusNotFound || strings.Contains(spoofResponse.Body.String(), "organization") {
		t.Fatalf("spoof status=%d body=%q", spoofResponse.Code, spoofResponse.Body.String())
	}
}

func hash(value string) []byte {
	digest := sha256.Sum256([]byte(value))
	return digest[:]
}
