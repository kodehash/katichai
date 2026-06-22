package safepath

import "testing"

func TestIsBlockedPath_blocked(t *testing.T) {
	cases := []string{
		".env",
		".env.local",
		".env.production",
		"foo/.env",
		"config/.env.development",
		"production.env",
		"staging.env",
		"config.env.production",
		"app/.env/file.txt",
		"secrets.pem",
		"id_rsa",
		"keys/server.key",
		"bundle.p12",
		"vault.pfx",
		"keepass.kdbx",
		".npmrc",
		"pkg/.npmrc",
		".pypirc",
		"secrets.json",
		"credentials",
		"etc/credentials.json",
		"home/.aws/credentials",
		"foo/application_default_credentials.json",
		"home/.ssh/id_rsa",
		".ssh/known_hosts",
	}
	for _, p := range cases {
		if !IsBlockedPath(p) {
			t.Errorf("expected blocked: %q", p)
		}
	}
}

func TestIsBlockedPath_allowed(t *testing.T) {
	cases := []string{
		"env.go",
		"environment.ts",
		"dotenv_test.go",
		"internal/config/config.go",
		"pkg/credentials/credentials.go",
		"auth/service.go",
		"README.md",
		"src/main.go",
	}
	for _, p := range cases {
		if IsBlockedPath(p) {
			t.Errorf("expected allowed: %q", p)
		}
	}
}

func TestIsBlockedPath_normalization(t *testing.T) {
	if !IsBlockedPath(`foo\bar\.env`) {
		t.Error("expected blocked windows-style path")
	}
	if IsBlockedPath(".") {
		t.Error("expected . allowed")
	}
}
