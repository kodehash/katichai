// Package safepath defines which repository-relative paths must never be read
// for context build or review (env files, common credential material, etc.).
//
// This is path/filename heuristics only; secrets renamed to look like source
// files cannot be detected here.
package safepath

import (
	"path/filepath"
	"strings"
)

// IsBlockedPath reports whether relPath (repository-relative) must be excluded
// from scanning, analysis, diff ingestion, and report file reads.
func IsBlockedPath(relPath string) bool {
	if relPath == "" {
		return false
	}
	p := filepath.ToSlash(filepath.Clean(relPath))
	if p == "." {
		return false
	}
	lower := strings.ToLower(p)
	segments := strings.Split(lower, "/")
	base := segments[len(segments)-1]
	if base == "" {
		return false
	}

	// Dotenv: .env, .env.local, .env.*, */.env, *.env, *.env.*
	if strings.HasPrefix(base, ".env") {
		return true
	}
	if strings.HasSuffix(base, ".env") {
		return true
	}
	if strings.Contains(base, ".env.") {
		return true
	}
	for _, seg := range segments {
		if seg == ".env" || strings.HasPrefix(seg, ".env.") {
			return true
		}
	}
	if strings.Contains(lower, "/.env/") {
		return true
	}

	// Key / cert material
	switch strings.ToLower(filepath.Ext(base)) {
	case ".pem", ".key", ".p12", ".pfx", ".kdbx":
		return true
	}
	switch base {
	case "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa":
		return true
	}

	// Common credential filenames (exact basename; does not match credentials.go)
	switch base {
	case ".npmrc", ".pypirc":
		return true
	case "secrets.json":
		return true
	case "credentials":
		return true
	case "credentials.json", "credentials.yml", "credentials.yaml":
		return true
	}

	// Cloud / tool default paths
	if strings.Contains(lower, ".aws/credentials") {
		return true
	}
	if strings.Contains(lower, "application_default_credentials.json") {
		return true
	}

	// SSH directory (private material and host hints)
	if strings.Contains(lower, "/.ssh/") || strings.HasPrefix(lower, ".ssh/") {
		return true
	}

	return false
}
