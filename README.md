# Katich AI

**Context-aware, AI-powered code review from your terminal.**

Katich AI reviews your code changes like a senior engineer — catching security vulnerabilities, architectural violations, duplicate logic, and unnecessary complexity. Works with OpenAI, Anthropic, or local Ollama models.

## Key Features

- **Security & Architecture Focus** — Critical issues prioritized by severity (security > breaking > architecture > performance)
- **Exact & Semantic Duplication** — Finds copy-pasted blocks and semantically similar logic via embeddings
- **Unnecessary Complexity Detection** — Flags over-engineered abstractions and excessive nesting
- **Fix Prompt Generation** — Produces a copy-pasteable prompt you can drop into Cursor, Copilot, or any AI assistant to fix findings
- **Multi-Language** — Go, Java, Python, TypeScript, JavaScript
- **Smart Chunking** — Automatically splits large diffs, processes in parallel, merges results. No token-limit errors.
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

That's it. For a complete guide to all commands, flags, configuration options, and output formats, see **[USAGE.md](USAGE.md)**.

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
