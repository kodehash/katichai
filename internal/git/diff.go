package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// DiffFile represents a file change in a diff
type DiffFile struct {
	Path      string
	OldPath   string // For renamed files
	Status    string // A (added), M (modified), D (deleted), R (renamed)
	Additions int
	Deletions int
	Patch     string // The actual diff content
}

// Diff represents a complete diff
type Diff struct {
	Files   []*DiffFile
	Commit  *Commit
	Summary string
}

// GetDiff returns the diff for a specific commit
func (r *Repository) GetDiff(ref string) (*Diff, error) {
	// Get commit info
	commit, err := r.GetCommit(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Get diff stats
	files, err := r.getDiffFiles(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	// Get summary
	summary, err := r.getDiffSummary(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff summary: %w", err)
	}

	return &Diff{
		Files:   files,
		Commit:  commit,
		Summary: summary,
	}, nil
}

// GetDiffRange returns the diff for a commit range
func (r *Repository) GetDiffRange(rangeSpec string) (*Diff, error) {
	files, err := r.getDiffFilesRange(rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	summary, err := r.getDiffSummaryRange(rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff summary: %w", err)
	}

	return &Diff{
		Files:   files,
		Summary: summary,
	}, nil
}

// getDiffFiles gets the list of changed files with stats for a single commit
func (r *Repository) getDiffFiles(ref string) ([]*DiffFile, error) {
	// Get file stats
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--numstat", "-r", ref)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff stats: %w", err)
	}

	files := make([]*DiffFile, 0)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		file := &DiffFile{
			Path: parts[2],
		}

		// Parse additions/deletions
		if parts[0] != "-" {
			fmt.Sscanf(parts[0], "%d", &file.Additions)
		}
		if parts[1] != "-" {
			fmt.Sscanf(parts[1], "%d", &file.Deletions)
		}

		// Get file status
		status, err := r.getFileStatus(ref, file.Path)
		if err == nil {
			file.Status = status
		}

		// Get patch for this file
		patch, err := r.getFilePatch(ref, file.Path)
		if err == nil {
			file.Patch = patch
		}

		files = append(files, file)
	}

	return files, nil
}

// getDiffFilesRange gets the list of changed files for a range
func (r *Repository) getDiffFilesRange(rangeSpec string) ([]*DiffFile, error) {
	cmd := exec.Command("git", "diff", "--numstat", rangeSpec)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff stats: %w", err)
	}

	files := make([]*DiffFile, 0)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		file := &DiffFile{
			Path: parts[2],
		}

		if parts[0] != "-" {
			fmt.Sscanf(parts[0], "%d", &file.Additions)
		}
		if parts[1] != "-" {
			fmt.Sscanf(parts[1], "%d", &file.Deletions)
		}

		// Get patch for this file
		patch, err := r.getFilePatchRange(rangeSpec, file.Path)
		if err == nil {
			file.Patch = patch
		}

		files = append(files, file)
	}

	return files, nil
}

// getFileStatus gets the status of a file (A, M, D, R)
func (r *Repository) getFileStatus(ref, filePath string) (string, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-status", "-r", ref)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filePath {
			return parts[0], nil
		}
	}

	return "M", nil // Default to modified
}

// getFilePatch gets the patch for a specific file
func (r *Repository) getFilePatch(ref, filePath string) (string, error) {
	// Get the actual diff
	diffCmd := exec.Command("git", "diff", fmt.Sprintf("%s^", ref), ref, "--", filePath)
	diffCmd.Dir = r.RootPath

	diffOutput, err := diffCmd.Output()
	if err != nil {
		return "", nil
	}

	return string(diffOutput), nil
}

// getFilePatchRange gets the patch for a file in a range
func (r *Repository) getFilePatchRange(rangeSpec, filePath string) (string, error) {
	cmd := exec.Command("git", "diff", rangeSpec, "--", filePath)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return "", nil
	}

	return string(output), nil
}

// getDiffSummary gets a summary of the diff
func (r *Repository) getDiffSummary(ref string) (string, error) {
	cmd := exec.Command("git", "diff", "--stat", fmt.Sprintf("%s^", ref), ref)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// getDiffSummaryRange gets a summary for a range
func (r *Repository) getDiffSummaryRange(rangeSpec string) (string, error) {
	cmd := exec.Command("git", "diff", "--stat", rangeSpec)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// GetFullDiff returns the complete diff output
func (r *Repository) GetFullDiff(ref string) (string, error) {
	cmd := exec.Command("git", "show", ref)
	cmd.Dir = r.RootPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get full diff: %w", err)
	}

	return string(output), nil
}
