# Katichai

**Context-aware, AI-assisted code review CLI tool**

Katichai prevents unnecessary AI-generated code, detects duplicated logic, enforces architectural patterns, and ensures high-quality engineering standards. It is designed to run locally or in CI/CD pipelines, providing engineering teams with an automated senior engineer's perspective.

## 🚀 Key Features

*   🧠 **Smart Change Classification**: Automatically detects if a change is `LOGIC`, `BOILERPLATE`, `REFACTOR`, or `DOCS` before deep review.
*   🔍 **Exact & Semantic Duplication**:
    *   **Exact**: Finds copy-pasted code blocks instantly using hashing.
    *   **Semantic**: Uses vector embeddings to find similar logic even if implemented differently.
*   ♻️ **Refactoring Suggestions**: Identifies "Reuse Candidates" (70-85% similarity) where you should use existing functions instead of writing new ones.
*   📉 **Code Drift Detection**: Flags inconsistencies in style, variable naming (e.g., `snake_case` in Go), and error handling patterns.
*   🤖 **AI Pattern Detection**: Identifies "AI Slop"—overly verbose, generic, or hallucinated code typical of LLM generation.
*   🏗️ **Architecture Enforcement**: Understands your framework (Spring, Next.js, FastAPI, Gin) and flags violations (e.g., calling DB from Controller).
*   🌐 **Multi-Language**: Native support for **Go, Java, Python, TypeScript, JavaScript**.
*   🔐 **Local & Secure**: Analyzes changes locally. Interaction with LLMs is configurable (OpenAI, Anthropic, or Local Ollama).
*   ⚡ **Intelligent Chunking with Parallel Processing**: Automatically splits large reviews into chunks when they exceed token limits, processes them in parallel (3x faster), and merges results seamlessly.
*   ⏱️ **Tokens-Per-Minute (TPM) Rate Limiting**: Proactively schedules chunk batches across 60-second windows to stay within your API tier's TPM limit — no more rate-limit errors mid-review.
*   📝 **GitHub Flavored Markdown (GFM) Reports**: Generate a clean, table-based `.md` report ideal for posting directly as a GitHub PR comment or Actions summary (`--gfm` flag).

## 🛠️ CLI Setup

### Prerequisites
*   **Go 1.22+** installed.
*   **Git** installed.
*   (Optional) **Ollama** for local embeddings/LLM.

### Installation

#### Option 1: Download Pre-built Binary (Recommended)

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_arm64.tar.gz | tar -xz
xattr -d com.apple.quarantine katich-darwin-arm64 2>/dev/null || true
sudo mv katich-darwin-arm64 /usr/local/bin/katich
```

**macOS (Intel):**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_amd64.tar.gz | tar -xz
xattr -d com.apple.quarantine katich-darwin-amd64 2>/dev/null || true
sudo mv katich-darwin-amd64 /usr/local/bin/katich
```

**Linux:**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_linux_amd64.tar.gz | tar -xz
sudo mv katich-linux-amd64 /usr/local/bin/katich
```

**Note for macOS:** If you see a security warning, the `xattr` command above removes the quarantine attribute. Alternatively, right-click the binary in Finder → "Open" → "Open" in the security dialog.

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/kodehash/katichai.git
cd katichai

# Install dependencies and build
go mod download
go build -o katich cmd/katich/main.go

# Move to path (optional)
mv katich /usr/local/bin/
```

### Configuration
Create a `.katich/config.yaml` in your project root (e.g. via `katich init`). The project name is automatically extracted from your Git repository during `katich init`. For a full list of options with comments, see **`.katich/config.example.yaml`** in the repo.

#### 1. Local LLM (Ollama) - Recommended for Privacy
```yaml
project_name: my-project  # Auto-extracted from Git repo during init

llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434  # Default

embeddings:
  provider: ollama
  model: nomic-embed-text
```

#### 2. OpenAI (GPT-4) - Direct API Key
```yaml
project_name: my-project

llm:
  provider: openai
  model: gpt-4o
  api_key: sk-...  # Or set via ENV: KATICH_LLM_API_KEY or OPENAI_API_KEY
  tokens_per_minute: 90000  # Match to your API tier (0 = disabled)

embeddings:
  provider: openai
  model: text-embedding-3-small
```

#### 3. Anthropic (Claude 3) - Direct API Key
```yaml
project_name: my-project

llm:
  provider: anthropic
  model: claude-3-opus-20240229
  api_key: sk-ant-...  # Or set via ENV: ANTHROPIC_API_KEY

embeddings:
  provider: openai  # Anthropic doesn't support embeddings yet, use OpenAI or Local
  model: text-embedding-3-small
```

#### 4. Centralized API Server (Recommended for Teams)
Fetch LLM API keys from a centralized server instead of storing them in config files.

**Environment Variables:**
```bash
export KATICH_API_SERVER_URL="https://api.example.com/api/v1/keys"  # Full API endpoint URL
export KATICH_API_TOKEN="your-api-token"
```

**Config File:**
```yaml
project_name: my-project  # Used to fetch the correct API key from server

llm:
  provider: openai
  model: gpt-4o
  # api_key: omitted - will be fetched from API server

api_server:
  enabled: true
  # url: from KATICH_API_SERVER_URL env var (required, must include full API path)
  token: "your-api-token"  # or from KATICH_API_TOKEN env var (env var takes precedence)
```

**How it works:**
- When `api_server.enabled: true`, Katich will fetch the LLM API key from the centralized server
- The API server receives the `project_name` and `provider` to return the appropriate key
- Fetched keys are cached for 1 hour to reduce API calls
- If the API fetch fails, the operation will error (no fallback to config)

**API Server Endpoint:**
The API server should implement:
- **Endpoint**: `POST {KATICH_API_SERVER_URL}` (must include the full path, e.g., `https://api.example.com/api/v1/keys`)
- **Headers**: 
  - `Authorization: Bearer {KATICH_API_TOKEN}`
  - `Content-Type: application/json`
- **Request Body**:
  ```json
  {
    "project_name": "my-project",
    "provider": "openai"
  }
  ```
- **Response**:
  ```json
  {
    "api_key": "sk-...",
    "expires_at": "2024-01-01T00:00:00Z"  // optional
  }
  ```

#### Environment Variable Overrides
All API keys can be overridden via environment variables (takes precedence over config):
- `KATICH_LLM_API_KEY` - Overrides `llm.api_key`
- `OPENAI_API_KEY` - Overrides `llm.api_key` when provider is `openai`
- `ANTHROPIC_API_KEY` - Overrides `llm.api_key` when provider is `anthropic`
- `KATICH_API_SERVER_URL` - Sets API server URL (required when `api_server.enabled: true`)
- `KATICH_API_TOKEN` - Sets API server authentication token (overrides `api_server.token`)

## ⚡ Quick Start

1.  **Initialize**: Set up configuration (creates `.katich/config.yaml`).
    ```bash
    katich init
    ```

2.  **Build context** (optional but recommended): Analyze the codebase and generate embeddings.
    ```bash
    katich context build
    ```
    Use `katich context build --incremental` (default) to only process changed files; use `--force` for a full rebuild. Use `--publish` to push `context.json` and `embeddings.json` to your Git remote (e.g. for centralized context).

3.  **Review changes**: Run a review on your current work.
    ```bash
    katich review latest
    ```

4.  **Review a specific diff**:
    ```bash
    katich review diff main..feature-branch
    ```

### Context source (local vs remote)
In `.katich/config.yaml`, `context.source` can be `local` (use `.katich/` on disk) or `remote` (fetch `context.json` and `embeddings.json` from your Git origin). When `remote`, review commands run `git fetch` and use the files from `context.remote.branch` and `context.remote.directory`. No GitHub token is required if you can push/pull with normal Git credentials.

### Embeddings (API)
When using OpenAI for embeddings (e.g. large repos), requests are sent in batches of 100 with a 2-minute timeout per batch and up to 3 retries with backoff. If you see timeouts, check network and API status; the tool continues without embeddings if generation fails so reviews still run (without semantic similarity).

## ⚡ Token Limits and Parallel Chunking

Katichai uses **optimistic chunking**: every review runs through the chunked pipeline so token limits and output size are always handled correctly, with no single-shot path that can hit context-window errors.

### How It Works
1. **Model limits**: Context window size is auto-detected from the model name (GPT-4, GPT-4o, Claude, Llama, Mistral, Gemma, etc.). Override with `llm.max_input_tokens` in config if needed.
2. **Chunking**: The review payload is split into self-contained chunks (security-related and high-risk files first, then by module/size). Each chunk gets a dynamic `max_tokens` so input + output never exceeds the model limit.
3. **Parallel processing**: Chunks are reviewed with up to 3 concurrent requests; TPM-aware scheduling groups chunks into 60-second windows so the total stays under `tokens_per_minute`.
4. **Merging**: Chunk results are deduplicated and merged; a final AI step normalizes summaries into one coherent report.

### Configuration
In `.katich/config.yaml`:

```yaml
llm:
  max_input_tokens: 0        # 0 = auto-detect from model name; set to override
  tokens_per_minute: 90000  # Your API tier's TPM limit (0 = disabled)
```

Common TPM values: OpenAI Tier 1 ~30K, Tier 2 ~90K; Claude standard ~40K. See `.katich/config.example.yaml` for all options.

### What You'll See
When multiple chunks or TPM windows are used:

```
📦 Split into 6 chunks for parallel review
🤖 Reviewing chunk 1/6...
🤖 Reviewing chunk 2/6...
⏳ Window 1/2 done. Waiting 47s before next batch to respect TPM limit...
🔄 Merging chunk results...
🔀 Normalizing 2 chunk summaries into a unified summary...
```

The final report format is the same whether the review was one chunk or many — no token limit or rate limit errors.

## 📝 Example Output

When running `katich review latest`:

```markdown
# Code Review Report

## Summary
The changes introduce a new `UserService` but duplicate logic from `AuthService` and violate the project's error handling patterns.

## 🚨 Critical Issues
- **[ARCH]** Direct database access in `UserController.go`. Use the Repository pattern.
- **[SECURITY]** Hardcoded secret detected in `config.go`.
- **[DUPLICATION]** `ValidateEmail` (User.go) is an exact duplicate of `AuthUtils.go:45`.

## ♻️ Refactoring Opportunities
- **Reuse Candidate**: `GenerateToken` is 82% similar to `SessionManager.CreateToken`. Consider reusing.

## ⚠️ Code Drift
- **Naming**: `user_id` (snake_case) used locally; project standard is `userID` (camelCase).

## Score: 65/100 (Request Changes)
```

## 💻 Code Setup (For Contributors)

If you want to contribute to Katichai:

1.  **Repository Structure**:
    *   `cmd/katich`: CLI entrypoints.
    *   `internal/analysis`: Static analysis, AST parsing, and Heuristics.
    *   `internal/review`: Review orchestration, Engine, Reviewer.
    *   `internal/llm`: LLM client, Prompts, Classifier.
    *   `internal/embeddings`: Vector storage (FAISS-like), Similarity search.

2.  **Running Tests**:
    ```bash
    go test ./...
    ```

3.  **Adding a New Detector**:
    *   Implement detection logic in `internal/analysis`.
    *   Register it in `internal/review/engine.go`.

## License
MIT
