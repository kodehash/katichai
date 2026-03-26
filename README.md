# Katich AI

**Context-aware, AI-powered code review from your terminal.**

Katich AI reviews your code changes like a senior engineer — catching security vulnerabilities, architectural violations, duplicate logic, and unnecessary complexity. Works with OpenAI, Anthropic, or local Ollama models.

## Key Features

- **Security & Architecture Focus** — Critical issues prioritized by severity (security > breaking > architecture > performance)
- **Exact & Semantic Duplication** — Finds copy-pasted blocks and semantically similar logic via embeddings
- **Unnecessary Complexity Detection** — Flags over-engineered abstractions and excessive nesting
- **Fix Prompt Generation** — Produces a copy-pasteable prompt you can drop into Cursor, Copilot, or any AI assistant to fix findings
- **Context-Aware Reviews** — Builds a repository context (`context.json` + embeddings) to improve review quality over time
- **Multi-Language** — Go, Java, Python, TypeScript, JavaScript
- **Smart Chunking + TPM Scheduling** — Automatically chunks big reviews and paces requests to avoid token-limit / rate-limit failures
- **Multiple Output Formats** — Console, HTML (interactive), GitHub Flavored Markdown (for PR comments), JSON

## Quick Start

### 1. Install

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_arm64.tar.gz | tar -xz
sudo mv katich-darwin-arm64 /usr/local/bin/katich
```

**macOS (Intel):**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_amd64.tar.gz | tar -xz
sudo mv katich-darwin-amd64 /usr/local/bin/katich
```

**Linux:**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_linux_amd64.tar.gz | tar -xz
sudo mv katich-linux-amd64 /usr/local/bin/katich
```

> **macOS security warning?** Run `xattr -d com.apple.quarantine /usr/local/bin/katich`

Or build from source: `git clone https://github.com/kodehash/katichai.git && cd katichai && go build -o katich cmd/katich/main.go`

### 2. Initialize

```bash
cd /path/to/your/project
katich init
```

This creates `.katich/config.yaml` with default settings.

### 3. Add your API key

Edit `.katich/config.yaml`:

```yaml
llm:
  provider: openai
  model: gpt-4o
  api_key: "sk-..."
```

Or set an environment variable instead:

```bash
export OPENAI_API_KEY="sk-..."
```

### 4. Build context (recommended)

```bash
katich context build
```

Analyzes your codebase and generates embeddings for semantic understanding. This step is optional but significantly improves review quality.

### 5. Review your code

```bash
katich review latest              # Review the latest commit
katich review diff main..feature  # Compare branches
katich review full                # Full repository audit
```

That's it. For the full reference, see **[USAGE.md](USAGE.md)**.

## Core Commands

```bash
katich init
katich context build [--force] [--publish]
katich review latest [--html] [--gfm] [--ai-detect] [--no-fix-prompt]
katich review diff main..feature [--html] [--gfm] [--ai-detect] [--no-fix-prompt]
katich review full [--html] [--gfm] [--ai-detect] [--no-fix-prompt]
katich doctor
katich version
```

## Review Flags (Most Used)

| Flag | Default | What it does |
|------|---------|--------------|
| `--html` | from config | Generate an interactive HTML report |
| `--gfm` | from config | Generate a GitHub Flavored Markdown report |
| `--ai-detect` | off | Enable AI-generated code detection (opt-in) |
| `--no-fix-prompt` | off | Disable fix prompt generation (fix prompt is on by default) |

Config equivalents live under `review:` in `.katich/config.yaml`. See [`.katich/config.example.yaml`](.katich/config.example.yaml).

## Configuration (Common Options)

Katich reads `.katich/config.yaml`. `katich init` generates it with defaults.

### LLM provider

```yaml
llm:
  provider: openai        # openai | anthropic | ollama
  model: gpt-4o
  api_key: "sk-..."       # or use OPENAI_API_KEY / ANTHROPIC_API_KEY
  tokens_per_minute: 90000  # 0 = disable TPM pacing
```

### Embeddings provider (recommended)

```yaml
embeddings:
  provider: openai        # openai | ollama
  model: text-embedding-3-small
```

### Reports and optional features

```yaml
review:
  generate_html: true
  generate_gfm: false
  detect_ai_code: false       # opt-in
  generate_fix_prompt: true   # on by default
```

## Context Build & Sharing (Local vs Remote)

By default, Katich uses **local** context stored in `.katich/` (`context.json` + `embeddings.json`). For teams, you can publish context to Git and have other clones fetch it.

```yaml
context:
  source: local   # local (default) | remote
  remote:
    branch: main
    directory: katich-ai-context
```

Publish context after building:

```bash
katich context build --publish
```

## Output Formats

- **Console (default)**: readable summary, grouped critical issues, suggestions, static analysis, fix prompt
- **HTML report**: interactive report in `.katich/reports/` (collapsible sections, copy buttons, navigation)
- **GFM report (`--gfm`)**: GitHub-friendly markdown for PR comments / Actions summaries (auto-truncates to 65k chars)
- **JSON / Markdown**: `--output json|markdown --output-file <path>`

## Token Limits, Chunking, and TPM

Katich uses **optimistic chunking**: all reviews run through a chunked pipeline so large diffs don't blow the model context window. If you set `llm.tokens_per_minute`, Katich also paces requests in 60-second windows to avoid provider rate limits.

```yaml
llm:
  max_input_tokens: 0       # 0 = auto-detect; set if you're using a custom model
  tokens_per_minute: 90000  # 0 = disabled
```

## CI / GitHub Actions (Example)

```yaml
- name: Katich review (GFM)
  run: |
    katich review diff ${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }} --gfm
    GFM_FILE=$(ls -t .katich/reports/*.md | head -1)
    echo "## Katich AI Code Review" >> $GITHUB_STEP_SUMMARY
    cat "$GFM_FILE" >> $GITHUB_STEP_SUMMARY
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

## Advanced: Centralized API Server (Teams)

If you don’t want API keys in local config files, Katich can fetch them from a central server. This is optional and intended for teams.

```bash
export KATICH_API_SERVER_URL="https://api.example.com/api/v1/keys"
export KATICH_API_TOKEN="your-api-token"
```

```yaml
api_server:
  enabled: true
  token: "your-api-token"   # or set KATICH_API_TOKEN (env var wins)
```

## Example Output

```
════════════════════════════════════════════════════════════
 🤖 AI CODE REVIEW REPORT
════════════════════════════════════════════════════════════

📌 SUMMARY
The changes introduce a new UserService but duplicate logic from AuthService
and violate the project's error handling patterns.

⚠️  CRITICAL ISSUES (3 found)
────────────────────────────────────────────────────────────

  🔴 SECURITY (2)

     1. Missing input validation on user email allows
        injection attacks
        📍 auth/service.go:45

     2. Hardcoded secret detected in configuration
        📍 config/api_config.go:12

  🟡 ARCHITECTURE (1)

     1. Direct database access in controller layer
        violates repository pattern
        📍 payment/controller.go:67

────────────────────────────────────────────────────────────

💡 SUGGESTIONS (2)
────────────────────────────────────────────────────────────

  1. Use parameterized queries for all database
     operations to prevent SQL injection

  2. Extract payment logic to a repository layer
     for better testability

────────────────────────────────────────────────────────────

🔧 FIX PROMPT (copy and paste into Cursor / AI assistant)
────────────────────────────────────────────────────────────
Fix the following issues found during code review.

[Security]
1. Fix at auth/service.go:45: Missing input validation...
2. Fix at config/api_config.go:12: Hardcoded secret...
...
────────────────────────────────────────────────────────────
```

## Supported LLM Providers

| Provider | Models | Notes |
|----------|--------|-------|
| **OpenAI** | gpt-4o, gpt-4, gpt-3.5-turbo | Recommended. Set `api_key` or `OPENAI_API_KEY` env var |
| **Anthropic** | claude-3-5-sonnet, claude-3-opus | Set `api_key` or `ANTHROPIC_API_KEY` env var |
| **Ollama** | llama3, mistral, gemma, etc. | Free, local, private. Requires [Ollama](https://ollama.ai) running locally |

## Full Documentation

- **Usage guide**: [USAGE.md](USAGE.md)
- **All config options (with comments)**: [`.katich/config.example.yaml`](.katich/config.example.yaml)

## Contributing

```
cmd/katich        CLI entrypoints
internal/analysis Static analysis, AST parsing, heuristics
internal/review   Review orchestration, engine, formatters
internal/llm      LLM client, prompts, classifier
internal/embeddings  Vector storage, similarity search
```

```bash
go test ./...
```

## License

MIT
