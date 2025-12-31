# Katich AI

**Context-aware AI code review tool**

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
- [Troubleshooting](#troubleshooting)

---

## Installation

### Option 1: Pre-built Binaries (Recommended)

Download the latest release for your platform:

**macOS (Intel)**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_amd64.tar.gz -o katich.tar.gz
tar -xzf katich.tar.gz
sudo mv katich /usr/local/bin/
```

**macOS (Apple Silicon)**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_arm64.tar.gz -o katich.tar.gz
tar -xzf katich.tar.gz
sudo mv katich /usr/local/bin/
```

**Note for macOS**: If you see a security warning, run:
```bash
xattr -d com.apple.quarantine /usr/local/bin/katich
```

**Linux**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_linux_amd64.tar.gz -o katich.tar.gz
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

Katich uses a configuration file at `.katich/config.yaml`. Run `katich init` to create it with defaults.

### Default Configuration

```yaml
project_name: your-project-name

llm:
  provider: openai
  model: gpt-4
  api_key: ""  # Add your OpenAI API key here
  max_input_tokens: 20000

embeddings:
  provider: local
  model: jina-code-v2

review:
  generate_html: true
  html_output_path: .katich/reports

analysis:
  max_function_length: 50
  complexity_threshold: 10
  similarity_threshold: 0.70
  min_function_lines: 5
  duplicate_threshold: 0.85
  ignore_trivial_patterns: true
  sampling:
    enabled: true
    max_files: 50
    context_lines: 2
    skip_generated: true
    skip_tests: false
    adaptive_budget: true
```

### Environment Variables

You can override configuration using environment variables:

- `KATICH_LLM_API_KEY` - LLM API key
- `OPENAI_API_KEY` - OpenAI API key (if provider is openai)
- `ANTHROPIC_API_KEY` - Anthropic API key (if provider is anthropic)
- `KATICH_API_SERVER_URL` - API server URL for key fetching
- `KATICH_API_TOKEN` - API server authentication token

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
katich context build
```

**What it does:**
- Analyzes all source files in the repository
- Generates code metrics and complexity analysis
- Creates embeddings for semantic code understanding
- Saves context to `.katich/context.json` and `.katich/embeddings.json`

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

**What it does:**
- Compares the latest commit with its parent
- Analyzes all changes in the commit
- Generates comprehensive review report
- Creates HTML report (if enabled)

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

Add to your CI pipeline:

```yaml
# GitHub Actions example
- name: Run Katich Review
  run: |
    katich review diff ${{ github.event.pull_request.base.sha }}..${{ github.event.pull_request.head.sha }}
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Custom Configuration

Override settings per review:

```bash
# Use different config file
katich review latest --config .katich/production.yaml

# Force HTML generation
katich review latest --html
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

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/kodehash/katichai/issues
- Documentation: https://github.com/kodehash/katichai

---

**Katich AI** - Making code reviews smarter, faster, and more comprehensive.

