package mcp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/review"
)

// ToolHandler is a function that executes a tool and returns text output.
type ToolHandler func(args map[string]interface{}) (string, error)

// ToolRegistry holds all MCP-exposed tools.
type ToolRegistry struct {
	defs     []toolDefinition
	handlers map[string]ToolHandler
}

// NewToolRegistry creates a registry with all katich tools registered.
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{handlers: make(map[string]ToolHandler)}
	r.registerAll()
	return r
}

// Definitions returns the tool schemas for tools/list.
func (r *ToolRegistry) Definitions() []toolDefinition {
	return r.defs
}

// Call dispatches a tool call by name.
func (r *ToolRegistry) Call(name string, args map[string]interface{}) (string, error) {
	h, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return h(args)
}

func (r *ToolRegistry) register(def toolDefinition, handler ToolHandler) {
	r.defs = append(r.defs, def)
	r.handlers[def.Name] = handler
}

func (r *ToolRegistry) registerAll() {
	r.register(toolDefinition{
		Name:        "katich_review_latest",
		Description: "Review the latest git commit. Returns findings prioritized by severity (security, breaking, architecture, performance) with a copy-pasteable fix prompt.",
		InputSchema: toolInputSchema{
			Type: "object",
			Properties: map[string]propertyDefinition{
				"format": {
					Type:        "string",
					Description: "Output format for the review report.",
					Enum:        []string{"terminal", "json", "markdown"},
				},
				"gfm": {
					Type:        "boolean",
					Description: "Also generate a GitHub Flavored Markdown report saved to .katich/reports/.",
				},
				"html": {
					Type:        "boolean",
					Description: "Also generate an HTML report saved to .katich/reports/.",
				},
				"ai_detect": {
					Type:        "boolean",
					Description: "Enable AI-generated code detection (slower).",
				},
				"no_fix_prompt": {
					Type:        "boolean",
					Description: "Disable fix prompt generation.",
				},
			},
		},
	}, handleReviewLatest)

	r.register(toolDefinition{
		Name:        "katich_review_diff",
		Description: "Review a specific git diff range (e.g. main..feature-branch). Use for PR reviews and branch comparisons.",
		InputSchema: toolInputSchema{
			Type: "object",
			Properties: map[string]propertyDefinition{
				"range": {
					Type:        "string",
					Description: "Git diff range, e.g. 'main..feature-branch' or 'HEAD~3..HEAD'.",
				},
				"format": {
					Type:        "string",
					Description: "Output format for the review report.",
					Enum:        []string{"terminal", "json", "markdown"},
				},
				"gfm": {
					Type:        "boolean",
					Description: "Also generate a GitHub Flavored Markdown report saved to .katich/reports/.",
				},
				"html": {
					Type:        "boolean",
					Description: "Also generate an HTML report saved to .katich/reports/.",
				},
				"ai_detect": {
					Type:        "boolean",
					Description: "Enable AI-generated code detection (slower).",
				},
			},
			Required: []string{"range"},
		},
	}, handleReviewDiff)

	r.register(toolDefinition{
		Name:        "katich_review_full",
		Description: "Review the entire repository. Performs a comprehensive audit of all tracked files using intelligent sampling. High token usage — use selectively.",
		InputSchema: toolInputSchema{
			Type: "object",
			Properties: map[string]propertyDefinition{
				"max_files": {
					Type:        "string",
					Description: "Maximum number of files to review (default 30). Katich picks the most important files.",
				},
				"format": {
					Type:        "string",
					Description: "Output format for the review report.",
					Enum:        []string{"terminal", "json", "markdown"},
				},
				"html": {
					Type:        "boolean",
					Description: "Also generate an HTML report saved to .katich/reports/.",
				},
			},
		},
	}, handleReviewFull)

	r.register(toolDefinition{
		Name:        "katich_context_build",
		Description: "Build codebase context and embeddings. Scans the repo, detects frameworks, parses ASTs, and generates embeddings. Improves review quality.",
		InputSchema: toolInputSchema{
			Type: "object",
			Properties: map[string]propertyDefinition{
				"force": {
					Type:        "boolean",
					Description: "Force a full rebuild, ignoring cache.",
				},
			},
		},
	}, handleContextBuild)

	r.register(toolDefinition{
		Name:        "katich_doctor",
		Description: "Check system requirements and configuration health. Verifies Git, config file, LLM API key, and embedding model.",
		InputSchema: toolInputSchema{
			Type:       "object",
			Properties: map[string]propertyDefinition{},
		},
	}, handleDoctor)

	r.register(toolDefinition{
		Name:        "katich_version",
		Description: "Display the installed katich version, git commit, and build date.",
		InputSchema: toolInputSchema{
			Type:       "object",
			Properties: map[string]propertyDefinition{},
		},
	}, handleVersion)
}

// --- Tool handlers ---

func handleReviewLatest(args map[string]interface{}) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	applyReviewFlags(cfg, args)

	repo, err := git.FindRepository()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	diff, err := repo.GetDiff("HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get latest diff: %w", err)
	}

	return runReview(cfg, repo, func(engine *review.ReviewEngine) (*review.ReviewReport, error) {
		return engine.Review(diff)
	}, args)
}

func handleReviewDiff(args map[string]interface{}) (string, error) {
	diffRange, _ := args["range"].(string)
	if diffRange == "" {
		return "", fmt.Errorf("'range' argument is required (e.g. main..feature-branch)")
	}

	cfg, err := loadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	applyReviewFlags(cfg, args)

	repo, err := git.FindRepository()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	diff, err := repo.GetDiffRange(diffRange)
	if err != nil {
		return "", fmt.Errorf("failed to get diff for %s: %w", diffRange, err)
	}

	return runReview(cfg, repo, func(engine *review.ReviewEngine) (*review.ReviewReport, error) {
		return engine.Review(diff, diffRange)
	}, args)
}

func handleReviewFull(args map[string]interface{}) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	applyReviewFlags(cfg, args)

	if v, ok := args["max_files"]; ok {
		if s, ok := v.(string); ok {
			var n int
			if _, err := fmt.Sscanf(s, "%d", &n); err == nil && n > 0 {
				cfg.Analysis.Sampling.MaxFiles = n
			}
		}
		if n, ok := v.(float64); ok && n > 0 {
			cfg.Analysis.Sampling.MaxFiles = int(n)
		}
	}

	repo, err := git.FindRepository()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	diff, err := repo.GetFullRepositoryDiff()
	if err != nil {
		return "", fmt.Errorf("failed to get repository files: %w", err)
	}

	return runReview(cfg, repo, func(engine *review.ReviewEngine) (*review.ReviewReport, error) {
		return engine.ReviewFullRepository(diff)
	}, args)
}

func handleContextBuild(args map[string]interface{}) (string, error) {
	force, _ := args["force"].(bool)

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "context", "build")
	if force {
		cmdArgs = append(cmdArgs, "--force")
	}

	return captureCommand(cmdArgs...)
}

func handleDoctor(args map[string]interface{}) (string, error) {
	return captureCommand("doctor")
}

func handleVersion(args map[string]interface{}) (string, error) {
	return captureCommand("version")
}

// --- Helpers ---

func loadConfig() (*config.Config, error) {
	return config.Load("")
}

func applyReviewFlags(cfg *config.Config, args map[string]interface{}) {
	if v, ok := args["html"].(bool); ok && v {
		cfg.Review.GenerateHTML = true
	}
	if v, ok := args["gfm"].(bool); ok && v {
		cfg.Review.GenerateGFM = true
	}
	if v, ok := args["ai_detect"].(bool); ok && v {
		cfg.Review.DetectAICode = true
	}
	if v, ok := args["no_fix_prompt"].(bool); ok && v {
		cfg.Review.GenerateFixPrompt = false
	}
}

func formatReport(report *review.ReviewReport, args map[string]interface{}) string {
	formatter := review.NewFormatter()
	format, _ := args["format"].(string)

	switch format {
	case "json":
		return formatter.FormatJSON(report)
	case "markdown":
		return formatter.FormatMarkdown(report)
	default:
		return formatter.FormatText(report)
	}
}

// runReview initialises the review engine, runs the provided reviewFn, and
// formats the result. Stdout noise from the engine is captured and discarded
// so the MCP response only contains the clean report.
func runReview(
	cfg *config.Config,
	repo *git.Repository,
	reviewFn func(*review.ReviewEngine) (*review.ReviewReport, error),
	args map[string]interface{},
) (string, error) {
	baseReviewer := review.NewReviewer(repo.RootPath, cfg)

	// Capture stdout from engine (it prints progress lines)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	engine, err := review.NewEngine(cfg, baseReviewer)
	if err != nil {
		w.Close()
		os.Stdout = old
		io.Copy(io.Discard, r)
		return "", fmt.Errorf("failed to initialize review engine: %w", err)
	}

	report, reviewErr := reviewFn(engine)

	w.Close()
	os.Stdout = old
	io.Copy(io.Discard, r)

	if reviewErr != nil {
		return "", fmt.Errorf("review failed: %w", reviewErr)
	}

	return formatReport(report, args), nil
}

// captureCommand runs a katich subcommand by re-invoking the binary and
// capturing its combined stdout+stderr. This is used for lightweight commands
// (doctor, version, context build) where re-executing the binary is simpler
// than pulling out the function internals.
func captureCommand(args ...string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot find katich executable: %w", err)
	}

	cmd := exec.Command(exe, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(buf.String())
		if output != "" {
			return "", fmt.Errorf("%s\n%s", err, output)
		}
		return "", err
	}

	return strings.TrimSpace(buf.String()), nil
}
