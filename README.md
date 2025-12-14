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

## 🛠️ CLI Setup

### Prerequisites
*   **Go 1.22+** installed.
*   **Git** installed.
*   (Optional) **Ollama** for local embeddings/LLM.

### Installation

```bash
# Clone the repository
git clone https://github.com/katichai/katich.git
cd katich

# Install dependencies and build
go mod download
go build -o katich cmd/katich/main.go

# Move to path (optional)
mv katich /usr/local/bin/
```

### Configuration
Create a `.katich/config.yaml` in your home directory or project root.

#### 1. Local LLM (Ollama) - Recommended for Privacy
```yaml
llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434  # Default

embeddings:
  provider: ollama
  model: nomic-embed-text
```

#### 2. OpenAI (GPT-4)
```yaml
llm:
  provider: openai
  model: gpt-4o
  api_key: sk-...  # Or set via ENV: KATICH_LLM_API_KEY

embeddings:
  provider: openai
  model: text-embedding-3-small
```

#### 3. Anthropic (Claude 3)
```yaml
llm:
  provider: anthropic
  model: claude-3-opus-20240229
  api_key: sk-ant-...

embeddings:
  provider: openai  # Anthropic doesn't support embeddings yet, use OpenAI or Local
  model: text-embedding-3-small
```

## ⚡ Quick Start

1.  **Initialize Context**: First, let Katichai learn your codebase.
    ```bash
    katich context build
    ```

2.  **Review Changes**: Run a review on your current work.
    ```bash
    katich review latest
    ```

3.  **Review Specific Diff**:
    ```bash
    katich review diff main..feature-branch
    ```

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
