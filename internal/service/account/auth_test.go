package account

import (
	"testing"

	"github.com/rehiy/libgo/secure"

	"isrvd/config"
)

func withAccountConfig(t *testing.T, oidc *config.OIDCConfig, passkey *config.PasskeyConfig, members map[string]*config.MemberConfig) {
	t.Helper()
	oldOIDC := config.OIDC
	oldPasskey := config.Passkey
	oldServer := config.Server
	oldMembers := config.Members
	config.OIDC = oidc
	config.Passkey = passkey
	config.Server = &config.ServerConfig{JWTSecret: "test-secret", JWTExpiration: 86400}
	config.Members = members
	t.Cleanup(func() {
		config.OIDC = oldOIDC
		config.Passkey = oldPasskey
		config.Server = oldServer
		config.Members = oldMembers
	})
}

func TestLoginRejectsPasswordWhenOIDCOnly(t *testing.T) {
	hash, err := secure.BcryptHash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	withAccountConfig(t,
		&config.OIDCConfig{Enabled: true, Only: true, IssuerURL: "https://idp.example.com", ClientID: "isrvd"},
		&config.PasskeyConfig{},
		map[string]*config.MemberConfig{"alice": {Username: "alice", Password: hash}},
	)

	_, err = (&Service{}).Login(LoginRequest{Username: "alice", Password: "secret"})
	if err == nil || err.Error() != "仅允许 OIDC 登录" {
		t.Fatalf("expected OIDC-only login error, got %v", err)
	}
}

func TestLoginPasswordWhenOIDCOnlyDisabled(t *testing.T) {
	hash, err := secure.BcryptHash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	withAccountConfig(t,
		&config.OIDCConfig{Enabled: true, IssuerURL: "https://idp.example.com", ClientID: "isrvd"},
		&config.PasskeyConfig{},
		map[string]*config.MemberConfig{"alice": {Username: "alice", Password: hash}},
	)

	resp, err := (&Service{}).Login(LoginRequest{Username: "alice", Password: "secret"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp == nil || resp.Token == "" || resp.Username != "alice" {
		t.Fatalf("unexpected login response: %#v", resp)
	}
}

func TestAuthInfoDisablesPasskeyInOIDCOnly(t *testing.T) {
	withAccountConfig(t,
		&config.OIDCConfig{Enabled: true, Only: true, IssuerURL: "https://idp.example.com", ClientID: "isrvd"},
		&config.PasskeyConfig{},
		map[string]*config.MemberConfig{"alice": {Username: "alice"}},
	)

	resp := (&Service{webAuthn: nil}).AuthInfo("")
	if !resp.OIDCEnabled || !resp.OIDCOnly {
		t.Fatalf("expected OIDC-only auth info, got %#v", resp)
	}
	if resp.PasskeyEnabled {
		t.Fatalf("expected passkey disabled in OIDC-only mode")
	}
}
