# Katich AI

**Context-aware AI code review tool**

```
██╗  ██╗ █████╗ ████████╗██╗ ██████╗██╗  ██╗     █████╗ ██╗
██║ ██╔╝██╔══██╗╚══██╔══╝██║██╔════╝██║  ██║    ██╔══██╗██║
█████╔╝ ███████║   ██║   ██║██║     ███████║    ███████║██║
██╔═██╗ ██╔══██║   ██║   ██║██║     ██╔══██║    ██╔══██║██║
██║  ██╗██║  ██║   ██║   ██║╚██████╗██║  ██║    ██║  ██║██║
╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝╚═╝  ╚═╝    ╚═╝  ╚═╝╚═╝
             context aware AI code review tool
```

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Commands](#commands)
  - [katich init](#katich-init)
  - [katich context build](#katich-context-build)
  - [katich review latest](#katich-review-latest)
  - [katich review diff](#katich-review-diff)
  - [katich review full](#katich-review-full)
  - [katich review file](#katich-review-file)
  - [katich doctor](#katich-doctor)
  - [katich version](#katich-version)
- [Output Formats](#output-formats)
- [Example Outputs](#example-outputs)
- [Best Practices](#best-practices)
- [Rate Limiting (TPM)](#rate-limiting-tpm)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Option 1: Pre-built Binaries (Recommended)

Download the latest release for your platform:

**macOS (Intel)**
```bash
curl -L https://github.com/kodehash/katichai-dist/releases/latest/download/katich_darwin_amd64.tar.gz -o katich.tar.gz
tar -xzf katich.tar.gz
sudo mv katich /usr/local/bin/
```

**macOS (Apple Silicon)**
```bash
curl -L https://github.com/kodehash/katichai-dist/releases/latest/download/katich_darwin_arm64.tar.gz -o katich.tar.gz
tar -xzf katich.tar.gz
sudo mv katich /usr/local/bin/
```

**Note for macOS**: If you see a security warning, run:
```bash
xattr -d com.apple.quarantine /usr/local/bin/katich
```

**Linux**
```bash
curl -L https://github.com/kodehash/katichai-dist/releases/latest/download/katich_linux_amd64.tar.gz -o katich.tar.gz
tar -xzf katich.tar.gz
sudo mv katich /usr/local/bin/
```

### Option 2: Build from Source

```bash
git clone https://github.com/kodehash/katichai.git
cd katichai
go build -o katich cmd/katich/main.go
sudo mv katich /usr/local/bin/
```

### Verify Installation

```bash
katich version
```

Expected output:
```
katich version v1.0.0-alpha.2
Git commit: abc1234
Build date: 2024-01-15
```

---

## Quick Start

1. **Navigate to your Git repository**
   ```bash
   cd /path/to/your/project
   ```

2. **Initialize Katich**
   ```bash
   katich init
   ```

3. **Add your OpenAI API key** to `.katich/config.yaml`:
   ```yaml
   llm:
     provider: openai
     model: gpt-4
     api_key: "sk-your-api-key-here"
   ```

4. **Build context** (optional but recommended)
   ```bash
   katich context build
   ```

5. **Review your code**
   ```bash
   katich review latest
   ```

---

## Configuration

Katich uses a configuration file at `.katich/config.yaml`. Run `katich init` to create it with defaults. For **all possible options** with descriptions, copy or reference **`.katich/config.example.yaml`** from the repository.

### Key options

| Section | Key | Purpose |
|--------|-----|---------|
| **llm** | provider, model, api_key | Which LLM to use (openai, anthropic, ollama) |
| **llm** | max_input_tokens | Override model context window (0 = auto-detect) |
| **llm** | tokens_per_minute | TPM rate limit for chunked reviews (0 = disabled) |
| **embeddings** | provider, model, api_key | Local (Ollama) or API (OpenAI) for embeddings |
| **context** | source | `local` (use .katich/) or `remote` (fetch from Git) |
| **context.remote** | branch, directory | Branch and directory for remote context |
| **review** | generate_html, generate_gfm | Output HTML and/or GFM reports |
| **analysis** | sampling.* | Max files, context lines, skip tests, etc. |

### Minimal examples

```yaml
# OpenAI
llm:
  provider: openai
  model: gpt-4o
  api_key: "sk-..."
  tokens_per_minute: 90000

# Ollama (local)
llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434

# Context: use remote repo for context/embeddings
context:
  source: remote
  remote:
    branch: main
    directory: katich-ai-context
```

### Environment Variables

You can override configuration using environment variables:

- `KATICH_LLM_API_KEY` — LLM API key
- `OPENAI_API_KEY` — OpenAI API key (if provider is openai)
- `ANTHROPIC_API_KEY` — Anthropic API key (if provider is anthropic)
- `KATICH_EMBEDDINGS_API_KEY` — Embeddings API key (else falls back to OPENAI_API_KEY)
- `KATICH_API_SERVER_URL` — API server URL for key fetching
- `KATICH_API_TOKEN` — API server authentication token

### Context source (local vs remote)

- **`context.source: local`** (default) — Use `context.json` and `embeddings.json` from the local `.katich/` directory. Run `katich context build` to create/update them.
- **`context.source: remote`** — Fetch both files from your Git remote (e.g. origin). Review commands will `git fetch` and read from `context.remote.branch` and `context.remote.directory` (default `main` and `katich-ai-context`). No GitHub token is required if you can push/pull with normal Git. Use `katich context build --publish` to push updated context from another clone.

### LLM Providers

**OpenAI** (Default)
```yaml
llm:
  provider: openai
  model: gpt-4
  api_key: "sk-..."
```

**Anthropic**
```yaml
llm:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
  api_key: "sk-ant-..."
```

**Ollama (Local)**
```yaml
llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434
```

---

## Commands

### katich init

Initialize Katich in the current directory. Creates `.katich/config.yaml` with default settings.

**Usage:**
```bash
katich init
```

**What it does:**
- Creates `.katich/` directory
- Generates `config.yaml` with defaults
- Extracts project name from Git repository

**Example Output:**
```
✅ Initialized katich configuration!
Created .katich/config.yaml with default settings (OpenAI).

Next steps:
  1. Add your OpenAI API key to config.yaml (llm.api_key)
  2. Run 'katich context build' to analyze your codebase
  3. Run 'katich review latest' to review the latest commit with previous commit
  4. Run 'katich review diff base_branch..new_branch' to perform review between two branches
```

---

### katich context build

Analyze your codebase and build context for better reviews. This is optional but recommended for improved review quality.

**Usage:**
```bash
katich context build                    # Incremental (default): only changed files
katich context build --force           # Full rebuild (ignore cache)
katich context build --publish         # Push context.json + embeddings.json to origin
katich context build -i                # Same as default (incremental)
katich context build -f                # Same as --force
```

**Flags:**
- `--incremental`, `-i` (default: true) — Only analyze/add embeddings for files that changed since the last build on the same branch. If HEAD hasn’t changed, prints "No changes found to perform context build" and exits.
- `--force`, `-f` — Ignore cached state and rebuild everything (context.json and embeddings).
- `--publish` — After building, commit and push `context.json` and `embeddings.json` into the configured remote branch (e.g. `main`) under the directory from `context.remote.directory` (default `katich-ai-context`). Uses normal Git (no GitHub token required if you can push).

**What it does:**
- Analyzes source files (all, or only changed when incremental)
- Generates code metrics and complexity analysis
- Creates embeddings for semantic code understanding (batches of 100 for API, with retries)
- Saves context to `.katich/context.json` and `.katich/embeddings.json`
- Build state (branch + commit) is stored in `.katich/.last_build_state` for incremental builds

**Example Output:**
```
🔍 Analyzing repository...

📊 Analysis Results:
  • Total files: 45
  • Total functions: 234
  • Total lines of code: 8,432
  • Average complexity: 4.2
  • Top complexity functions: 5

Most Complex Functions:
  1. processPayment (complexity: 18, 45 lines)
  2. validateUserInput (complexity: 15, 32 lines)
  3. generateReport (complexity: 12, 67 lines)
  4. authenticateUser (complexity: 11, 28 lines)
  5. calculateTax (complexity: 10, 25 lines)

🧠 Generating embeddings...
  Using provider: local
  ✅ Generated 234 embeddings
  💾 Saved to .katich/embeddings.json

Architectural patterns:
  • MVC Pattern
  • Repository Pattern
  • Service Layer

Configuration files found:
  • package.json
  • go.mod
  • docker-compose.yml
```

---

### katich review latest

Review the latest commit by comparing it with the previous commit.

**Usage:**
```bash
katich review latest
```

**Flags:**
```bash
katich review latest --html   # Force HTML report generation
katich review latest --gfm    # Generate a GitHub Flavored Markdown report
```

**What it does:**
- Compares the latest commit with its parent
- Analyzes all changes in the commit
- Generates comprehensive review report
- Creates HTML report (if `review.generate_html: true` or `--html` flag)
- Creates GFM report (if `review.generate_gfm: true` or `--gfm` flag)

**Example Output:**
```
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║   ██╗  ██╗ █████╗ ████████╗██╗ ██████╗██╗  ██╗              ║
║   ██║ ██╔╝██╔══██╗╚══██╔══╝██║██╔════╝██║  ██║              ║
║   █████╔╝ ███████║   ██║   ██║██║     ███████║              ║
║   ██╔═██╗ ██╔══██║   ██║   ██║██║     ██╔══██║              ║
║   ██║  ██╗██║  ██║   ██║   ██║╚██████╗██║  ██║              ║
║   ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝╚═╝  ╚═╝              ║
║                                                              ║
║              Context-aware AI code review tool              ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝

📊 Sampling Report:
  • Total files changed: 12
  • Filtered: 3 (hidden: 1, generated: 2)
  • Reviewing: 9 files

🔴 High Priority Files:
  • src/auth/service.go (Risk: 85) - security, core logic
  • src/payment/processor.go (Risk: 72) - security, high complexity

🤖 Querying LLM for review...

════════════════════════════════════════════════════════════
 🤖 AI CODE REVIEW REPORT
════════════════════════════════════════════════════════════

📊 Tokens: 4523 input + 1847 output = 6370 total

📌 SUMMARY
This commit introduces authentication improvements and payment processing updates.
Key concerns include potential security vulnerabilities in the authentication flow
and missing input validation in payment processing.

⚠️  CRITICAL ISSUES
🔴 [SECURITY] Missing input validation on user email in auth/service.go:45
   📍 auth/service.go:45
🟡 [ARCHITECTURE] Payment processor violates repository pattern in payment/processor.go:123
   📍 payment/processor.go:123
🔴 [SECURITY] Potential SQL injection risk in user query in auth/service.go:67
   📍 auth/service.go:67

💡 SUGGESTIONS
• Add input validation for email format
• Consider using parameterized queries for database operations
• Extract payment logic to repository layer

🔄 DUPLICATE CODE
  • Found 2 duplicate code blocks (see HTML report for details)

📊 STATIC ANALYSIS (informational, not affecting score)
  • High Complexity: 3 files
  • Long Functions: 2 files
  • Naming Issues: 1 file

🤖 AI-GENERATED CODE ANALYSIS
  • auth/service.go (85%)
  • payment/processor.go (72%)

🔧 UNNECESSARY COMPLEXITY
  [Code] Function has excessive nesting levels (complexity: 18) (Score: 75% - High)
    📍 auth/service.go:45 - Function: authenticateUser
    📝 Reasoning: This function uses 5 levels of nesting when 2 would suffice. The logic could be simplified by extracting helper functions and using early returns. The current implementation requires 40 lines to accomplish what could be done in 15 lines.
    💡 Suggestion: Consider refactoring to reduce nesting depth by extracting functions or using early returns

✅ HTML report generated: .katich/reports/review-20240115-143022.html
```

---

### katich review diff

Review changes between two commits or branches.

**Usage:**
```bash
katich review diff <range>
```

**Examples:**
```bash
# Compare two branches
katich review diff main..feature-branch

# Compare two commits
katich review diff HEAD~3..HEAD

# Compare with specific commit
katich review diff abc1234..def5678

# Compare current branch with main
katich review diff main..HEAD
```

**Flags:**
```bash
katich review diff main..feature --html  # Force HTML report
katich review diff main..feature --gfm   # Generate GFM report (ideal for PR comments)
```

**What it does:**
- Analyzes all changes between the specified range
- Reviews added, modified, and deleted files
- Generates comprehensive review report

**Example Output:**
```
╔══════════════════════════════════════════════════════════════╗
║                    Katich AI                                 ║
║        Context-aware AI code review tool                      ║
╚══════════════════════════════════════════════════════════════╝

📊 Sampling Report:
  • Total files changed: 25
  • Filtered: 5 (package_management: 2, generated: 2, hidden: 1)
  • Reviewing: 20 files

🤖 Querying LLM for review...

[Review output similar to katich review latest]
```

---

### katich review full

Perform a comprehensive review of the entire repository (all tracked files). Uses intelligent sampling to handle large repositories.

**Usage:**
```bash
katich review full
```

**What it does:**
- Reviews all tracked files in the repository
- Uses intelligent sampling (up to 100 files)
- Provides extensive coverage of architecture, security, and code quality
- No token limit for comprehensive analysis

**Example Output:**
```
╔══════════════════════════════════════════════════════════════╗
║                    Katich AI                                 ║
║        Context-aware AI code review tool                      ║
╚══════════════════════════════════════════════════════════════╝

📊 Sampling Report:
  • Total files in repository: 156
  • Filtered: 48 (package_management: 12, generated: 15, hidden: 8, test: 13)
  • Reviewing: 50 files

🔴 High Priority Files:
  • src/auth/service.go (Risk: 95) - security, core logic, high complexity
  • src/payment/processor.go (Risk: 88) - security, high complexity
  • src/database/connection.go (Risk: 82) - security, core logic
  • src/api/middleware.go (Risk: 75) - security, core logic
  • src/utils/crypto.go (Risk: 73) - security

🤖 Querying LLM for comprehensive repository review...

[Comprehensive review output covering entire codebase]
```

**When to use:**
- Initial codebase audit
- Pre-release comprehensive review
- Architecture assessment
- Security audit

---

### katich review file

Review a specific file for code quality, duplicates, and AI-generated patterns.

**Usage:**
```bash
katich review file <path>
```

**Examples:**
```bash
katich review file src/auth/service.go
katich review file internal/review/engine.go
```

**What it does:**
- Analyzes a single file
- Detects code quality issues
- Identifies duplicate code
- Detects AI-generated patterns

**Example Output:**
```
╔══════════════════════════════════════════════════════════════╗
║                    Katich AI                                 ║
║        Context-aware AI code review tool                      ║
╚══════════════════════════════════════════════════════════════╝

📌 Analyzing: src/auth/service.go

📊 File Analysis:
  • Functions: 8
  • Lines of code: 234
  • Average complexity: 6.2
  • AI-generated: 15% (low confidence)

⚠️  CRITICAL ISSUES
🔴 [SECURITY] Missing input validation on user email
   📍 src/auth/service.go:45
🟡 [COMPLEXITY] High cyclomatic complexity (18)
   📍 src/auth/service.go:67

💡 SUGGESTIONS
• Add input validation for email format
• Consider breaking down complex functions
```

---

### katich doctor

Check system requirements and configuration.

**Usage:**
```bash
katich doctor
```

**What it does:**
- Verifies Git installation
- Checks Go installation
- Validates Git repository
- Checks configuration file
- Verifies LLM provider setup
- Checks embedding model configuration

**Example Output:**
```
🔍 Running system diagnostics...

Git installation:        ✅ 2.42.0
Go installation:         ✅ Found
Git repository:          ✅ Found (branch: main)
Configuration file:     ✅ Found
LLM Provider:            ✅ Configured (openai)
Embedding model:         ✅ Configured (jina-code-v2)

💡 Tip: Create a .katich/config.yaml file to configure LLM and embedding settings
```

---

### katich version

Display version information.

**Usage:**
```bash
katich version
```

**Example Output:**
```
katich version v1.0.0-alpha.2
Git commit: 9977071
Build date: 2024-01-15
```

---

## Output Formats

### Console Output (Default)

Human-readable text output with emojis and formatting. Shows:
- Summary
- Critical issues
- Suggestions
- Static analysis summary
- AI-generated code analysis summary
- Unnecessary complexity issues

### HTML Report

Comprehensive interactive HTML report with:
- Dashboard with metrics
- Detailed critical issues
- Suggestions with formatting
- Duplicate code with side-by-side diffs
- AI-generated code analysis with function-level breakdown
- Full static analysis details
- File sampling information
- Unnecessary complexity section

**Location:** `.katich/reports/review-YYYYMMDD-HHMMSS.html`

**Features:**
- Collapsible sections
- Syntax highlighting
- File navigation
- Responsive design

### GitHub Flavored Markdown (GFM) Report

A clean, table-based Markdown report designed to be posted directly as a GitHub PR comment or pasted into a GitHub Actions job summary. Maximum 65,000 characters.

**Enable via config:**
```yaml
review:
  generate_gfm: true
  gfm_output_path: .katich/reports
```

**Enable via flag (one-off):**
```bash
katich review latest --gfm
katich review diff main..feature --gfm
```

**Location:** `.katich/reports/review-YYYYMMDD-HHMMSS.md`

**What's included:**
- Overall score badge and risk level
- File coverage summary
- Critical issues table with GitHub file links
- Suggestions and unnecessary complexity tables
- Duplicate code links (URLs only, no code snippets)
- DB/ORM query review findings

**What's excluded (to keep it GitHub-friendly):**
- No code snippets (only links in `owner/repo/blob/sha/file#Lnn` format)
- No dashboard summary section
- No token usage display
- Content is truncated gracefully if it would exceed 65,000 characters

**Ideal for GitHub Actions:**
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

### JSON Output

Machine-readable JSON format for integration with other tools.

**Usage:**
```bash
katich review latest --output json --output-file review.json
```

### Markdown Output

Markdown format suitable for PR comments or documentation.

**Usage:**
```bash
katich review latest --output markdown --output-file review.md
```

---

## Example Outputs

### Example 1: Security Issue Detection

```
⚠️  CRITICAL ISSUES
🔴 [SECURITY] SQL injection vulnerability in user query
   📍 database/user_repository.go:123
   Description: Raw SQL query uses string concatenation with user input.
   Recommendation: Use parameterized queries or prepared statements.

🔴 [SECURITY] Hardcoded API key found
   📍 config/api_config.go:45
   Description: API key is hardcoded in source code.
   Recommendation: Move to environment variables or secure key management.
```

### Example 2: Architecture Issue

```
🟡 [ARCHITECTURE] Violation of repository pattern
   📍 service/payment_service.go:67
   Description: Service layer directly accesses database instead of using repository.
   Impact: Tight coupling, difficult to test, violates separation of concerns.
   Recommendation: Extract database access to repository layer.
```

### Example 3: Performance Issue

```
🟡 [PERFORMANCE] N+1 query problem detected
   📍 service/order_service.go:89
   Description: Loop executes database query for each iteration.
   Impact: Significant performance degradation with large datasets.
   Recommendation: Use batch loading or eager loading.
```

### Example 4: Unnecessary Complexity

```
🔧 UNNECESSARY COMPLEXITY
  [Architectural] File path suggests multiple layers of abstraction (3 layers detected) (Score: 60% - Medium)
    📍 src/service/manager/handler/processor.go:1
    📝 Reasoning: This codebase uses a Service -> Manager -> Handler -> Processor chain for a simple CRUD operation. A single service layer would be sufficient. The multiple layers add indirection without providing any benefit.
    💡 Suggestion: Consider if all these layers are necessary - simpler architecture may suffice
```

---

## Best Practices

### 1. Regular Reviews

Run reviews frequently, especially before:
- Creating pull requests
- Merging to main branch
- Releasing new versions

### 2. Context Building

Run `katich context build` periodically (weekly/monthly) to keep embeddings up to date:
```bash
katich context build
```

### 3. Full Repository Reviews

Use `katich review full` for:
- Initial codebase audits
- Pre-release comprehensive checks
- Quarterly architecture reviews

### 4. Review Specific Changes

For focused reviews, use:
```bash
# Review your feature branch
katich review diff main..feature-branch

# Review latest changes
katich review latest
```

### 5. HTML Reports

Always enable HTML reports for detailed analysis:
```yaml
review:
  generate_html: true
  html_output_path: .katich/reports
```

### 6. GFM Reports for CI/CD

Enable GFM reports when running in GitHub Actions to post results directly as PR comments or job summaries:
```yaml
review:
  generate_gfm: true
  gfm_output_path: .katich/reports
```
Or use the `--gfm` flag for a one-off run without changing config.

### 6. Sampling Configuration

Adjust sampling for your repository size:
```yaml
analysis:
  sampling:
    max_files: 50  # Increase for larger repos
    skip_tests: false  # Set to true to exclude test files
```

### 7. Security Focus

The tool automatically focuses on security vulnerabilities. Ensure your API key has access to models that support security analysis (e.g., GPT-4, Claude 3.5 Sonnet).

---

## Troubleshooting

### Issue: "failed to fetch API key"

**Solution:**
- Check that `llm.api_key` is set in `.katich/config.yaml`
- Or set `OPENAI_API_KEY` environment variable
- If using API server, ensure `api_server.enabled: true` and URL/token are correct

### Issue: "Not in a Git repository"

**Solution:**
- Ensure you're in a Git repository directory
- Run `git init` if needed
- Make at least one commit before running reviews

### Issue: "LLM review failed"

**Possible causes:**
- Invalid API key
- Network connectivity issues
- Rate limiting from provider
- Insufficient API credits

**Solution:**
- Verify API key is correct
- Check network connection
- Wait and retry if rate limited
- Check API account credits

### Issue: HTML report not generated

**Solution:**
- Ensure `review.generate_html: true` in config
- Check write permissions for `.katich/reports/` directory
- Verify disk space available

### Issue: GFM report not generated

**Solution:**
- Ensure `review.generate_gfm: true` in config, or pass the `--gfm` flag
- Check write permissions for `.katich/reports/` directory

### Issue: "429 rate limit" / "tokens per minute exceeded"

**Solution:**
- Set `llm.tokens_per_minute` in config to match your API tier's TPM limit
- Katich will automatically pace chunk batches to stay within the limit
- If already set, lower the value slightly to add a buffer
- Set to `0` to disable scheduling (useful for high-tier API accounts)

```yaml
llm:
  tokens_per_minute: 90000  # Lower if still hitting rate limits
```

### Issue: "context deadline exceeded" / "Client.Timeout exceeded" when generating embeddings

**Cause:** The OpenAI embeddings API call timed out (e.g. many files in one go or slow network).

**Solution:**
- Katich now sends embeddings in **batches of 100** with a **2-minute timeout** per batch and **up to 3 retries** with backoff. Updating to the latest release should resolve most timeouts.
- If it still happens: check network and OpenAI status; use a smaller repo or run `katich context build --incremental` so only changed files need new embeddings.
- If embeddings fail, the tool continues without them (reviews still run, but without semantic similarity).

### Issue: "This model's maximum context length is X tokens... you requested Y"

**Cause:** The model’s total context window was exceeded (input + output).

**Solution:**
- Katich uses **optimistic chunking**: every review is chunked and each chunk gets a dynamic `max_tokens`, so this error should not occur with current versions. If you still see it, set an explicit limit in config so chunking uses the correct window: `llm.max_input_tokens: <your_model_limit>` (e.g. 8192 for older GPT-4). Ensure you’re on the latest release.

### Issue: "No issues found" but you expect issues

**Possible causes:**
- Changes are only comments/formatting
- Issues filtered out by sampling
- LLM response truncated (check token usage)

**Solution:**
- Review HTML report for full details
- Check sampling report to see which files were reviewed
- Increase `max_files` in sampling config
- For full reviews, use `katich review full`

### Issue: macOS security warning

**Solution:**
```bash
xattr -d com.apple.quarantine /usr/local/bin/katich
```

Or allow in System Settings > Privacy & Security.

---

## Advanced Usage

### CI/CD Integration

**Basic usage:**
```yaml
# GitHub Actions example
- name: Run Katich Review
  run: |
    katich review diff ${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }}
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

**With GFM report posted to job summary:**
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

**With TPM rate limiting for shared API keys:**
```yaml
- name: Run Katich Review
  run: |
    katich review diff ${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }} --gfm
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    # tokens_per_minute is set in .katich/config.yaml; lower it if CI hits rate limits
```

### Custom Configuration

Override settings per review:

```bash
# Use different config file
katich review latest --config .katich/production.yaml

# Force HTML generation
katich review latest --html

# Generate GFM report (GitHub Flavored Markdown)
katich review latest --gfm

# Combine flags
katich review diff main..feature --html --gfm
```

### API Server Integration

For centralized key management:

```yaml
api_server:
  enabled: true
  url: https://your-api-server.com/api/v1/keys
  token: your-auth-token
```

The API server should return API keys based on `project_name` and `provider` from your config.

---

---

## Token limits and chunking

### Optimistic chunking

Every review runs through the **chunked** pipeline. There is no single large request that can exceed the model’s context window: each chunk gets a dynamic `max_tokens` so input + output stay within the limit. This avoids "context length exceeded" errors regardless of diff size.

### Model context window

Context window size is **auto-detected** from the model name (e.g. GPT-4 8K, GPT-4o 128K, Claude 200K, Llama 3.1 128K). To override (e.g. for an unknown or custom model), set in config:

```yaml
llm:
  max_input_tokens: 8192  # 0 = auto-detect (default)
```

### Rate limiting (TPM)

Tokens Per Minute (TPM) is the rate limit enforced by LLM API providers. Katich schedules chunks into 60-second windows so the total tokens per minute stay under your limit:

1. **Token estimation** — Each chunk’s token cost (input + estimated output) is calculated.
2. **Window assignment** — Chunks are packed into windows under `tokens_per_minute`.
3. **Pacing** — After each window, Katich waits out the remainder of the 60-second slot before the next batch.
4. **Concurrency** — Within a window, up to 3 requests run in parallel.

```yaml
llm:
  tokens_per_minute: 90000  # Set to 0 to disable TPM limiting
```

Provider limits: **OpenAI** https://platform.openai.com/settings/organization/limits — **Anthropic** https://console.anthropic.com/settings/limits

### What you’ll see

```
📦 Split into 6 chunks for parallel review
🤖 Reviewing chunk 1/6...
🤖 Reviewing chunk 2/6...
⏳ Window 1/2 done. Waiting 47s before next batch to respect TPM limit...
🔀 Normalizing 2 chunk summaries into a unified summary...
```

---

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/kodehash/katichai/issues
- Documentation: https://github.com/kodehash/katichai

---

**Katich AI** - Making code reviews smarter, faster, and more comprehensive.

