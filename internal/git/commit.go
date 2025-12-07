package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Commit represents a Git commit
type Commit struct {
	Hash      string
	Author    string
	Email     string
	Date      time.Time
	Message   string
	ShortHash string
}

// GetLatestCommit returns the most recent commit
func (r *Repository) GetLatestCommit() (*Commit, error) {
	return r.GetCommit("HEAD")
}

// GetCommit returns information about a specific commit
func (r *Repository) GetCommit(ref string) (*Commit, error) {
	// Format: hash|author|email|timestamp|subject
	format := "%H|%an|%ae|%at|%s"
	
	cmd := exec.Command("git", "log", "-1", fmt.Sprintf("--format=%s", format), ref)
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}
	
	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) != 5 {
		return nil, fmt.Errorf("unexpected git log output format")
	}
	
	// Parse timestamp
	timestamp := strings.TrimSpace(parts[3])
	var date time.Time
	if timestamp != "" {
		var unixTime int64
		fmt.Sscanf(timestamp, "%d", &unixTime)
		date = time.Unix(unixTime, 0)
	}
	
	commit := &Commit{
		Hash:      parts[0],
		Author:    parts[1],
		Email:     parts[2],
		Date:      date,
		Message:   parts[4],
		ShortHash: parts[0][:7],
	}
	
	return commit, nil
}

// GetCommitRange returns commits in a range
func (r *Repository) GetCommitRange(rangeSpec string) ([]*Commit, error) {
	format := "%H|%an|%ae|%at|%s"
	
	cmd := exec.Command("git", "log", fmt.Sprintf("--format=%s", format), rangeSpec)
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit range: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]*Commit, 0, len(lines))
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			continue
		}
		
		var date time.Time
		var unixTime int64
		fmt.Sscanf(parts[3], "%d", &unixTime)
		date = time.Unix(unixTime, 0)
		
		commit := &Commit{
			Hash:      parts[0],
			Author:    parts[1],
			Email:     parts[2],
			Date:      date,
			Message:   parts[4],
			ShortHash: parts[0][:7],
		}
		
		commits = append(commits, commit)
	}
	
	return commits, nil
}

// GetChangedFiles returns the list of files changed in a commit
func (r *Repository) GetChangedFiles(ref string) ([]string, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", ref)
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0, len(lines))
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	
	return files, nil
}

// GetFileContent returns the content of a file at a specific commit
func (r *Repository) GetFileContent(ref, filePath string) (string, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", ref, filePath))
	cmd.Dir = r.RootPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}
	
	return string(output), nil
}
