package system

import (
	"testing"

	"isrvd/config"
)

func withSystemConfig(t *testing.T, tha *config.THAConfig, oidc *config.OIDCConfig) {
	t.Helper()
	oldTHA := config.THA
	oldOIDC := config.OIDC
	config.THA = tha
	config.OIDC = oidc
	t.Cleanup(func() {
		config.THA = oldTHA
		config.OIDC = oldOIDC
	})
}

func TestValidateAuthConfigRejectsIncompleteOIDCOnly(t *testing.T) {
	withSystemConfig(t, &config.THAConfig{}, &config.OIDCConfig{})

	err := validateAuthConfig(AllConfig{
		OIDC: &config.OIDCConfig{Enabled: true, Only: true, IssuerURL: "https://idp.example.com"},
	})
	if err == nil {
		t.Fatalf("expected incomplete OIDC-only config to fail")
	}
}

func TestValidateAuthConfigRejectsOIDCOnlyWithHeaderAuth(t *testing.T) {
	withSystemConfig(t, &config.THAConfig{}, &config.OIDCConfig{})

	err := validateAuthConfig(AllConfig{
		THA:  &config.THAConfig{Enabled: true, HeaderName: "X-Username"},
		OIDC: &config.OIDCConfig{Enabled: true, Only: true, IssuerURL: "https://idp.example.com", ClientID: "isrvd"},
	})
	if err == nil {
		t.Fatalf("expected OIDC-only and header auth conflict to fail")
	}
}

func TestValidateAuthConfigAcceptsOIDCOnly(t *testing.T) {
	withSystemConfig(t, &config.THAConfig{}, &config.OIDCConfig{})

	err := validateAuthConfig(AllConfig{
		OIDC: &config.OIDCConfig{Enabled: true, Only: true, IssuerURL: "https://idp.example.com", ClientID: "isrvd"},
	})
	if err != nil {
		t.Fatalf("validate OIDC-only config: %v", err)
	}
}
