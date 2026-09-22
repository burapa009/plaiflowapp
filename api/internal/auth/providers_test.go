package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProviderAuthorizationRequests(t *testing.T) {
	attempt := Attempt{State: "state", Nonce: "nonce", CodeChallenge: "challenge"}
	tests := []struct {
		name     string
		provider Provider
		scope    string
	}{
		{
			name: "line",
			provider: NewLINEProvider(ProviderConfig{
				ClientID: "line-client", ClientSecret: "line-secret", RedirectURI: "https://app.example/api/auth/line/callback",
			}),
			scope: "openid profile",
		},
		{
			name: "google",
			provider: NewGoogleProvider(ProviderConfig{
				ClientID: "google-client", ClientSecret: "google-secret", RedirectURI: "https://app.example/api/auth/google/callback",
			}),
			scope: "openid email profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, err := tt.provider.AuthorizationURL(attempt)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := url.Parse(location)
			if err != nil {
				t.Fatal(err)
			}
			query := parsed.Query()
			if query.Get("response_type") != "code" || query.Get("state") != "state" || query.Get("nonce") != "nonce" {
				t.Fatalf("authorization query = %v", query)
			}
			if query.Get("code_challenge") != "challenge" || query.Get("code_challenge_method") != "S256" {
				t.Fatalf("PKCE query = %v", query)
			}
			if query.Get("scope") != tt.scope {
				t.Fatalf("scope = %q", query.Get("scope"))
			}
			encoded := strings.ToLower(query.Encode())
			for _, forbidden := range []string{"drive", "sheets", "offline", "include_granted_scopes"} {
				if strings.Contains(encoded, forbidden) {
					t.Fatalf("authorization request contains %q: %s", forbidden, encoded)
				}
			}
		})
	}
}

func TestGoogleCallbackValidatesSignedIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	nonce := "google-nonce"
	idToken := signedGoogleToken(t, key, map[string]any{
		"iss": googleIssuer, "sub": "google-subject", "aud": "google-client", "azp": "google-client",
		"exp": time.Now().Add(time.Minute).Unix(), "nonce": nonce, "email": "google@example.test",
		"email_verified": true, "name": "Plai Google", "picture": "https://example.test/google",
	})
	var tokenForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			tokenForm = r.Form
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "discard-google", "id_token": idToken})
		case "/jwks":
			n := base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes())
			e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kid": "test-key", "kty": "RSA", "alg": "RS256", "n": n, "e": e}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewGoogleProvider(ProviderConfig{
		ClientID: "google-client", ClientSecret: "google-secret", RedirectURI: "https://app.example/api/auth/google/callback",
	}).(*oauthProvider)
	provider.tokenEndpoint = server.URL + "/token"
	provider.verifyEndpoint = server.URL + "/jwks"
	provider.client = server.Client()

	identity, err := provider.Exchange(context.Background(), "google-code", "google-verifier", nonce)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Issuer != googleIssuer || identity.Subject != "google-subject" || !identity.EmailVerified {
		t.Fatalf("identity = %+v", identity)
	}
	if tokenForm.Get("code_verifier") != "google-verifier" || tokenForm.Get("client_secret") != "google-secret" {
		t.Fatalf("token form = %v", tokenForm)
	}
	if _, err := provider.Exchange(context.Background(), "google-code", "google-verifier", "wrong-nonce"); err == nil {
		t.Fatal("wrong nonce accepted")
	}
}

func signedGoogleToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestLINECallbackExchangesAndValidatesIdentity(t *testing.T) {
	nonce := "expected-nonce"
	var tokenForm, verifyForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/token":
			tokenForm = r.Form
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "discard-me", "id_token": "line-id-token"})
		case "/verify":
			verifyForm = r.Form
			_ = json.NewEncoder(w).Encode(map[string]any{
				"iss": lineIssuer, "sub": "line-subject", "aud": "line-client", "exp": time.Now().Add(time.Minute).Unix(),
				"nonce": nonce, "name": "Plai", "picture": "https://example.test/picture", "email": "line@example.test",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := NewLINEProvider(ProviderConfig{
		ClientID: "line-client", ClientSecret: "line-secret", RedirectURI: "https://app.example/api/auth/line/callback",
	}).(*oauthProvider)
	provider.tokenEndpoint = server.URL + "/token"
	provider.verifyEndpoint = server.URL + "/verify"
	provider.client = server.Client()

	identity, err := provider.Exchange(context.Background(), "callback-code", "pkce-verifier", nonce)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Issuer != lineIssuer || identity.Subject != "line-subject" || identity.Email != "line@example.test" {
		t.Fatalf("identity = %+v", identity)
	}
	if tokenForm.Get("grant_type") != "authorization_code" || tokenForm.Get("code") != "callback-code" || tokenForm.Get("code_verifier") != "pkce-verifier" {
		t.Fatalf("token form = %v", tokenForm)
	}
	if verifyForm.Get("id_token") != "line-id-token" || verifyForm.Get("client_id") != "line-client" || verifyForm.Get("nonce") != nonce {
		t.Fatalf("verify form = %v", verifyForm)
	}

	provider.verifyEndpoint = server.URL + "/verify-wrong-nonce"
	if _, err := provider.Exchange(context.Background(), "callback-code", "pkce-verifier", nonce); err == nil {
		t.Fatal("invalid LINE verification response accepted")
	}
}

func TestNewAttemptUsesIndependentSecretsAndPKCE(t *testing.T) {
	first, err := NewAttempt()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewAttempt()
	if err != nil {
		t.Fatal(err)
	}
	if first.State == first.Nonce || first.State == first.CodeVerifier || first.State == second.State {
		t.Fatal("attempt secrets are not independent")
	}
	if len(first.State) < 43 || len(first.Nonce) < 43 || len(first.CodeVerifier) < 43 || len(first.CodeChallenge) < 43 {
		t.Fatalf("attempt secrets are too short: %+v", first)
	}
}
