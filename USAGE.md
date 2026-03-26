# Katich AI — Usage Guide

Complete reference for installing, configuring, and using Katich AI.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [LLM Providers](#llm-providers)
  - [Embeddings](#embeddings)
  - [Environment Variables](#environment-variables)
- [Commands](#commands)
  - [katich init](#katich-init)
  - [katich context build](#katich-context-build)
  - [katich review latest](#katich-review-latest)
  - [katich review diff](#katich-review-diff)
  - [katich review full](#katich-review-full)
  - [katich review file](#katich-review-file)
  - [katich doctor](#katich-doctor)
  - [katich version](#katich-version)
- [Review Flags Reference](#review-flags-reference)
- [Output Formats](#output-formats)
  - [Console](#console-output-default)
  - [HTML Report](#html-report)
  - [GFM Report](#gfm-report)
  - [JSON / Markdown](#json--markdown-output)
- [Review Features](#review-features)
  - [Fix Prompt](#fix-prompt)
  - [AI-Generated Code Detection](#ai-generated-code-detection)
  - [Long Function Handling](#long-function-handling)
  - [Security Focus](#security-focus)
- [Token Limits & Chunking](#token-limits--chunking)
- [CI/CD Integration](#cicd-integration)
- [Advanced Configuration](#advanced-configuration)
  - [Context Source (local vs remote)](#context-source-local-vs-remote)
  - [Centralized API Server](#centralized-api-server)
  - [Sampling Configuration](#sampling-configuration)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Option 1: Pre-built Binary (Recommended)

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

> **macOS security warning?** Run: `xattr -d com.apple.quarantine /usr/local/bin/katich`

### Option 2: Build from Source

Requires Go 1.22+.

```bash
git clone https://github.com/kodehash/katichai.git
cd katichai
go build -o katich cmd/katich/main.go
sudo mv katich /usr/local/bin/
```

### Verify

```bash
katich version
```

---

## Quick Start

```bash
# 1. Go to your project
cd /path/to/your/project

# 2. Initialize (creates .katich/config.yaml)
katich init

# 3. Add your API key to .katich/config.yaml
#    (or set OPENAI_API_KEY environment variable)

# 4. Build context (optional, improves review quality)
katich context build

# 5. Review your code
katich review latest
```

After `katich init`, open `.katich/config.yaml` and set your LLM provider and API key. The simplest setup:

```yaml
llm:
  provider: openai
  model: gpt-4o
  api_key: "sk-..."
```

That's all you need. Everything else has sensible defaults.

---

## Configuration

Configuration lives in `.katich/config.yaml`. Run `katich init` to generate it. For every possible option with descriptions, see [`.katich/config.example.yaml`](.katich/config.example.yaml).

### LLM Providers

**OpenAI** (recommended):
```yaml
llm:
  provider: openai
  model: gpt-4o
  api_key: "sk-..."
  tokens_per_minute: 90000   # Match your API tier (0 = no limit)
```

**Anthropic:**
```yaml
llm:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
  api_key: "sk-ant-..."
```

**Ollama** (local, free, private):
```yaml
llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434
```

> **Note:** Anthropic does not support embeddings. When using Anthropic as LLM provider, use OpenAI or Ollama for embeddings.

### Embeddings

Embeddings enable semantic code understanding (duplicate detection, similarity analysis). They are optional but recommended.

```yaml
# OpenAI embeddings (API)
embeddings:
  provider: openai
  model: text-embedding-3-small

# Ollama embeddings (local)
embeddings:
  provider: ollama
  model: nomic-embed-text
```

### Environment Variables

All API keys can be set via environment variables instead of (or in addition to) the config file. Environment variables take precedence.

| Variable | Purpose |
|----------|---------|
| `OPENAI_API_KEY` | OpenAI API key (LLM and embeddings) |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `KATICH_LLM_API_KEY` | Override LLM API key for any provider |
| `KATICH_EMBEDDINGS_API_KEY` | Override embeddings API key |

---

## Commands

### katich init

Create `.katich/config.yaml` with default settings. Project name is auto-detected from Git.

```bash
katich init
```

```
✅ Initialized katich configuration!
Created .katich/config.yaml with default settings (OpenAI).

Next steps:
  1. Add your OpenAI API key to config.yaml (llm.api_key)
  2. Run 'katich context build' to analyze your codebase
  3. Run 'katich review latest' to review the latest commit
```

---

### katich context build

Analyze your codebase and build embeddings for semantic code understanding. Optional but recommended.

```bash
katich context build                # Incremental (default) — only changed files
katich context build --force        # Full rebuild, ignore cache
katich context build --publish      # Push context to Git remote after building
```

| Flag | Short | Description |
|------|-------|-------------|
| `--incremental` | `-i` | Only process files changed since last build (default) |
| `--force` | `-f` | Full rebuild — ignore cache and regenerate everything |
| `--publish` | | Commit and push `context.json` + `embeddings.json` to the remote branch |

**What it does:**
- Scans source files and generates code metrics
- Creates embeddings for semantic understanding (with token-aware batching and retries)
- Saves to `.katich/context.json` and `.katich/embeddings.json`
- Tracks build state in `.katich/.last_build_state` for incremental builds

---

### katich review latest

Review the latest commit by comparing it to its parent.

```bash
katich review latest
katich review latest --html --gfm
katich review latest --ai-detect
katich review latest --no-fix-prompt
```

See [Review Flags Reference](#review-flags-reference) for all available flags.

---

### katich review diff

Review changes between two branches or commits.

```bash
katich review diff main..feature-branch
katich review diff HEAD~3..HEAD
katich review diff abc1234..def5678
```

All [review flags](#review-flags-reference) apply here too.

---

### katich review full

Comprehensive review of the entire repository. Uses intelligent sampling for large codebases.

```bash
katich review full
```

Best for:
- Initial codebase audits
- Pre-release comprehensive reviews
- Architecture assessments
- Security audits

---

### katich review file

Review a single file.

```bash
katich review file src/auth/service.go
```

---

### katich doctor

Check that your system and configuration are set up correctly.

```bash
katich doctor
```

```
🔍 Running system diagnostics...

Git installation:        ✅ 2.42.0
Go installation:         ✅ Found
Git repository:          ✅ Found (branch: main)
Configuration file:      ✅ Found
LLM Provider:            ✅ Configured (openai)
Embedding model:         ✅ Configured (text-embedding-3-small)
```

---

### katich version

```bash
katich version
```

---

## Review Flags Reference

All flags below work with `katich review latest`, `katich review diff`, and `katich review full`.

| Flag | Default | Description |
|------|---------|-------------|
| `--html` | from config | Generate an interactive HTML report |
| `--gfm` | from config | Generate a GitHub Flavored Markdown report |
| `--ai-detect` | `false` | Enable AI-generated code detection |
| `--no-fix-prompt` | `false` | Disable fix prompt generation |
| `--output <format>` | `text` | Output format: `text`, `json`, `markdown` |
| `--output-file <path>` | | Write output to a file |

**Config equivalents** (set in `.katich/config.yaml` under `review:`):

```yaml
review:
  generate_html: true          # same as --html
  html_output_path: .katich/reports
  generate_gfm: false          # same as --gfm
  gfm_output_path: .katich/reports
  detect_ai_code: false        # same as --ai-detect
  generate_fix_prompt: true    # opposite of --no-fix-prompt
```

CLI flags override config values for that run.

---

## Output Formats

### Console Output (Default)

Human-readable text with ANSI colors. Shows:
- Summary
- Critical issues (grouped by category: Security, Breaking, Architecture, Performance)
- Suggestions (numbered, word-wrapped)
- Duplicate code summary
- Static analysis (informational)
- Unnecessary complexity
- Fix prompt

### HTML Report

Interactive, self-contained HTML file with collapsible sections, syntax highlighting, and a sidebar.

**Location:** `.katich/reports/review-YYYYMMDD-HHMMSS.html`

Enable via config (`review.generate_html: true`) or flag (`--html`).

### GFM Report

Clean Markdown report designed for GitHub PR comments or Actions job summaries. Maximum 65,000 characters with graceful truncation.

**Location:** `.katich/reports/review-YYYYMMDD-HHMMSS.md`

Enable via config (`review.generate_gfm: true`) or flag (`--gfm`).

Includes:
- Risk level and file coverage
- Critical issues table with file links
- Suggestions, complexity, and duplicate findings
- Fix prompt (collapsible `<details>` block)

### JSON / Markdown Output

```bash
katich review latest --output json --output-file review.json
katich review latest --output markdown --output-file review.md
```

---

## Review Features

### Fix Prompt

After every review, Katich generates a structured, copy-pasteable **fix prompt** you can drop directly into Cursor, Copilot, Antigravity, or any AI coding assistant.

The prompt includes:
- All critical issues grouped by priority (security first)
- Suggestions as actionable items
- Safety constraints (don't break functionality, don't remove public APIs, run tests, etc.)

**Enabled by default.** Appears in console output, GFM reports (collapsible), and HTML reports (with a "Copy" button).

To disable:
```bash
katich review latest --no-fix-prompt
```
Or in config:
```yaml
review:
  generate_fix_prompt: false
```

### AI-Generated Code Detection

Detects patterns typical of AI-generated code (boilerplate, hallucinated APIs, over-engineered abstractions).

**Disabled by default** — since most code now involves AI assistance, this is opt-in to avoid noise.

To enable:
```bash
katich review latest --ai-detect
```
Or in config:
```yaml
review:
  detect_ai_code: true
```

### Long Function Handling

Function-length / line-count concerns (e.g., "function is too long") appear only in the **Static Analysis** section as informational findings. They are intentionally excluded from Critical Issues and Suggestions, which focus on security, architecture, performance, and breaking changes.

### Security Focus

Katich automatically prioritizes security vulnerabilities. Critical issues are sorted by severity: **Security > Breaking > Architecture > Performance**. Ensure your API key has access to capable models (GPT-4, Claude 3.5 Sonnet, etc.) for best results.

---

## Token Limits & Chunking

### How It Works

Every review runs through an **optimistic chunking** pipeline — there is no single large request that can exceed the model's context window.

1. **Auto-detect limits** — Context window size is detected from the model name (GPT-4 8K, GPT-4o 128K, Claude 200K, Llama 3.1 128K, etc.)
2. **Split into chunks** — The review payload is split into self-contained chunks (security and high-risk files first)
3. **Dynamic output tokens** — Each chunk gets a calculated `max_tokens` so input + output stays within the limit
4. **Parallel processing** — Up to 3 concurrent requests per window
5. **Merge results** — Chunk results are deduplicated and merged into one coherent report

### Rate Limiting (TPM)

LLM providers enforce tokens-per-minute limits. Katich schedules chunk batches into 60-second windows to stay under your limit:

```yaml
llm:
  tokens_per_minute: 90000   # Your API tier's TPM limit (0 = disabled)
  max_input_tokens: 0        # 0 = auto-detect from model name
```

Common TPM values: OpenAI Tier 1 ~30K, Tier 2 ~90K; Anthropic standard ~40K.

When chunking is active, you'll see:
```
📦 Split into 6 chunks for parallel review
🤖 Reviewing chunk 1/6...
🤖 Reviewing chunk 2/6...
⏳ Window 1/2 done. Waiting 47s before next batch to respect TPM limit...
🔀 Normalizing 2 chunk summaries into a unified summary...
```

---

## CI/CD Integration

### GitHub Actions

**Basic:**
```yaml
- name: Run Katich Review
  run: katich review diff ${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }}
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

**With GFM report in job summary:**
```yaml
- name: Run Katich Review
  run: |
    katich review diff ${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }} --gfm
    GFM_FILE=$(ls -t .katich/reports/*.md | head -1)
    echo "## Katich AI Code Review" >> $GITHUB_STEP_SUMMARY
    cat "$GFM_FILE" >> $GITHUB_STEP_SUMMARY
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

> **Tip:** Set `llm.tokens_per_minute` in your config to avoid rate-limit errors when multiple CI jobs share the same API key.

---

## Advanced Configuration

These options are for teams or specialized setups. New users can skip this section.

### Context Source (local vs remote)

By default, context is stored locally in `.katich/`. For teams, you can share context via Git:

```yaml
context:
  source: remote              # "local" (default) or "remote"
  remote:
    branch: main
    directory: katich-ai-context
```

- **`local`** — Uses `.katich/context.json` and `.katich/embeddings.json` on disk
- **`remote`** — Fetches both files from your Git remote. Review commands run `git fetch` and read from the configured branch/directory. No GitHub token needed if you can push/pull with normal Git credentials.

Use `katich context build --publish` to push updated context to the remote.

### Centralized API Server

For teams that manage API keys centrally instead of storing them in config files.

```yaml
api_server:
  enabled: true
  token: "your-auth-token"     # Or set KATICH_API_TOKEN env var
```

```bash
export KATICH_API_SERVER_URL="https://api.example.com/api/v1/keys"
export KATICH_API_TOKEN="your-api-token"
```

When enabled, Katich fetches the LLM API key from your server using the `project_name` and `provider` from config. Keys are cached for 1 hour.

**API server endpoint spec:**
- `POST {KATICH_API_SERVER_URL}` with `Authorization: Bearer {token}`
- Request body: `{ "project_name": "...", "provider": "openai" }`
- Response: `{ "api_key": "sk-...", "expires_at": "..." }`

| Environment Variable | Purpose |
|---------------------|---------|
| `KATICH_API_SERVER_URL` | Full URL to the key-fetching endpoint (required) |
| `KATICH_API_TOKEN` | Bearer token for authentication |

### Sampling Configuration

Control how many files are reviewed and which ones are skipped:

```yaml
analysis:
  sampling:
    max_files: 50          # Max files to review (increase for larger repos)
    skip_tests: false      # Set to true to exclude test files
```

---

## Troubleshooting

### "failed to fetch API key"

- Check that `llm.api_key` is set in `.katich/config.yaml`
- Or set `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` environment variable
- If using API server: ensure `api_server.enabled: true` and URL/token are correct

### "Not in a Git repository"

- Ensure you're inside a Git repository
- Run `git init` and make at least one commit before running reviews

### "LLM review failed"

- Verify your API key is correct and has credits
- Check network connectivity
- If rate-limited, wait and retry or set `tokens_per_minute` in config

### HTML or GFM report not generated

- Enable in config (`review.generate_html: true` / `review.generate_gfm: true`) or pass `--html` / `--gfm`
- Check write permissions for `.katich/reports/`

### "429 rate limit" / "tokens per minute exceeded"

Set `llm.tokens_per_minute` to match your API tier's limit. Katich will pace chunk batches automatically.

```yaml
llm:
  tokens_per_minute: 90000   # Lower if still hitting rate limits
```

### "context deadline exceeded" when generating embeddings

Katich batches embeddings with a 2-minute timeout per batch and retries up to 3 times. If it persists:
- Check your network and OpenAI API status
- Use `katich context build --incremental` so only changed files need new embeddings
- Embeddings are optional — reviews still work without them (minus semantic similarity)

### "token limit" when generating embeddings

Katich automatically splits oversized batches in half (up to 3 recursions). If a single snippet exceeds the limit, emergency truncation is applied. No action needed unless you see repeated failures.

### "This model's maximum context length is X tokens"

This should not occur with current versions (optimistic chunking handles it). If you still see it:
```yaml
llm:
  max_input_tokens: 8192   # Set explicitly for your model
```

### macOS security warning

```bash
xattr -d com.apple.quarantine /usr/local/bin/katich
```

Or allow in **System Settings > Privacy & Security**.

---

## Support

- GitHub Issues: https://github.com/kodehash/katichai/issues
- Full config reference: [`.katich/config.example.yaml`](.katich/config.example.yaml)
