package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repository represents a Git repository
type Repository struct {
	RootPath string
}

// FindRepository finds the Git repository root from the current directory
func FindRepository() (*Repository, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find git root
	rootPath, err := findGitRoot(cwd)
	if err != nil {
		return nil, err
	}

	return &Repository{
		RootPath: rootPath,
	}, nil
}

// findGitRoot finds the root of the git repository
func findGitRoot(startPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = startPath
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("not a git repository: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to find git root: %w", err)
	}

	rootPath := strings.TrimSpace(string(output))
	return rootPath, nil
}

// IsGitInstalled checks if git is installed and available
func IsGitInstalled() bool {
	cmd := exec.Command("git", "--version")
	err := cmd.Run()
	return err == nil
}

// GetGitVersion returns the installed git version
func GetGitVersion() (string, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

// GetCurrentBranch returns the current branch name
func (r *Repository) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	
	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// GetRemoteURL returns the remote URL for the repository
func (r *Repository) GetRemoteURL(remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	
	cmd := exec.Command("git", "remote", "get-url", remote)
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	
	url := strings.TrimSpace(string(output))
	return url, nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func (r *Repository) HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetRelativePath returns the path relative to the repository root
func (r *Repository) GetRelativePath(absPath string) (string, error) {
	relPath, err := filepath.Rel(r.RootPath, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	return relPath, nil
}
